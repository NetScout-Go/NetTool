import { useEffect, useCallback, useRef, useState } from 'react'

/**
 * WebSocket message types for plugin execution streaming
 */
export const StreamMessageType = {
  // Lifecycle messages from server
  EXECUTION_STARTED: 'execution_started',
  EXECUTION_PROGRESS: 'execution_progress',
  EXECUTION_COMPLETED: 'execution_completed',
  EXECUTION_FAILED: 'execution_failed',
  EXECUTION_CANCELLED: 'execution_cancelled',
  
  // Client request messages
  CANCEL_EXECUTION: 'cancel_execution',
  GET_STATUS: 'get_execution_status',
  SUBSCRIBE: 'subscribe',
  UNSUBSCRIBE: 'unsubscribe',
}

/**
 * Execution status enum
 */
export const ExecutionStatus = {
  PENDING: 'pending',
  RUNNING: 'running',
  COMPLETED: 'completed',
  FAILED: 'failed',
  CANCELLED: 'cancelled',
}

/**
 * Hook for streaming plugin execution via WebSocket
 * @param {Object} options - Hook options
 * @param {Function} options.onProgress - Callback for progress updates
 * @param {Function} options.onComplete - Callback when execution completes
 * @param {Function} options.onError - Callback for errors
 * @param {boolean} options.autoConnect - Auto-connect to WebSocket (default: true)
 * @returns {Object} Hook state and methods
 */
export function usePluginStream({ 
  onProgress, 
  onComplete, 
  onError,
  autoConnect = true 
} = {}) {
  const wsRef = useRef(null)
  const [isConnected, setIsConnected] = useState(false)
  const [executions, setExecutions] = useState({})
  const subscribedRef = useRef(new Set())
  const callbacksRef = useRef({ onProgress, onComplete, onError })

  // Keep callbacks up to date
  useEffect(() => {
    callbacksRef.current = { onProgress, onComplete, onError }
  }, [onProgress, onComplete, onError])

  // WebSocket connection
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws/plugins`
    
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onopen = () => {
      setIsConnected(true)
      // Resubscribe to all executions
      subscribedRef.current.forEach(executionId => {
        ws.send(JSON.stringify({
          type: StreamMessageType.SUBSCRIBE,
          executionId,
        }))
      })
    }

    ws.onclose = () => {
      setIsConnected(false)
      // Attempt reconnection after delay
      setTimeout(() => {
        if (subscribedRef.current.size > 0) {
          connect()
        }
      }, 3000)
    }

    ws.onerror = (error) => {
      console.error('Plugin WebSocket error:', error)
      callbacksRef.current.onError?.(error)
    }

    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        handleMessage(message)
      } catch (error) {
        console.error('Error parsing WebSocket message:', error)
      }
    }
  }, [])

  // Disconnect WebSocket
  const disconnect = useCallback(() => {
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    setIsConnected(false)
  }, [])

  // Handle incoming messages
  const handleMessage = useCallback((message) => {
    const { type, executionId, data } = message

    setExecutions(prev => ({
      ...prev,
      [executionId]: data,
    }))

    switch (type) {
      case StreamMessageType.EXECUTION_STARTED:
      case StreamMessageType.EXECUTION_PROGRESS:
        callbacksRef.current.onProgress?.(executionId, data)
        break

      case StreamMessageType.EXECUTION_COMPLETED:
        callbacksRef.current.onComplete?.(executionId, data)
        // Auto-unsubscribe on completion
        unsubscribe(executionId)
        break

      case StreamMessageType.EXECUTION_FAILED:
      case StreamMessageType.EXECUTION_CANCELLED:
        callbacksRef.current.onError?.(executionId, data)
        unsubscribe(executionId)
        break

      default:
        break
    }
  }, [])

  // Subscribe to execution updates
  const subscribe = useCallback((executionId) => {
    subscribedRef.current.add(executionId)
    
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: StreamMessageType.SUBSCRIBE,
        executionId,
      }))
    } else {
      connect()
    }
  }, [connect])

  // Unsubscribe from execution updates
  const unsubscribe = useCallback((executionId) => {
    subscribedRef.current.delete(executionId)
    
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: StreamMessageType.UNSUBSCRIBE,
        executionId,
      }))
    }

    // Disconnect if no more subscriptions
    if (subscribedRef.current.size === 0) {
      disconnect()
    }
  }, [disconnect])

  // Cancel an execution
  const cancelExecution = useCallback((executionId) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: StreamMessageType.CANCEL_EXECUTION,
        executionId,
      }))
    }
  }, [])

  // Get execution status
  const getExecutionStatus = useCallback((executionId) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({
        type: StreamMessageType.GET_STATUS,
        executionId,
      }))
    }
  }, [])

  // Auto-connect on mount if enabled
  useEffect(() => {
    if (autoConnect && subscribedRef.current.size > 0) {
      connect()
    }

    return () => {
      disconnect()
    }
  }, [autoConnect, connect, disconnect])

  return {
    isConnected,
    executions,
    connect,
    disconnect,
    subscribe,
    unsubscribe,
    cancelExecution,
    getExecutionStatus,
  }
}

/**
 * Hook for a single plugin execution
 * @param {string} executionId - The execution ID to track
 * @returns {Object} Execution state and controls
 */
export function usePluginExecution(executionId) {
  const [status, setStatus] = useState(ExecutionStatus.PENDING)
  const [progress, setProgress] = useState(null)
  const [result, setResult] = useState(null)
  const [error, setError] = useState(null)

  const { 
    isConnected, 
    subscribe, 
    unsubscribe, 
    cancelExecution 
  } = usePluginStream({
    onProgress: (id, data) => {
      if (id === executionId) {
        setStatus(data.status)
        setProgress(data.progress)
      }
    },
    onComplete: (id, data) => {
      if (id === executionId) {
        setStatus(ExecutionStatus.COMPLETED)
        setResult(data.result)
        setProgress(null)
      }
    },
    onError: (id, data) => {
      if (id === executionId) {
        setStatus(data.status || ExecutionStatus.FAILED)
        setError(data.error)
        setProgress(null)
      }
    },
  })

  // Subscribe when executionId is set
  useEffect(() => {
    if (executionId) {
      subscribe(executionId)
      return () => unsubscribe(executionId)
    }
  }, [executionId, subscribe, unsubscribe])

  const cancel = useCallback(() => {
    if (executionId) {
      cancelExecution(executionId)
    }
  }, [executionId, cancelExecution])

  const isRunning = status === ExecutionStatus.PENDING || status === ExecutionStatus.RUNNING
  const isComplete = status === ExecutionStatus.COMPLETED
  const isFailed = status === ExecutionStatus.FAILED || status === ExecutionStatus.CANCELLED

  return {
    status,
    progress,
    result,
    error,
    isConnected,
    isRunning,
    isComplete,
    isFailed,
    cancel,
  }
}

export default usePluginStream
