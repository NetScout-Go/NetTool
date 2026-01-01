import { useState, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import { motion } from 'framer-motion'
import { 
  ArrowLeft, 
  Play, 
  Settings, 
  RefreshCw,
  AlertCircle,
  CheckCircle,
  Clock,
  Copy,
  Download,
  Repeat,
  StopCircle
} from 'lucide-react'
import { usePlugin } from '../context/PluginContext'
import { Card, CardHeader, Button, Badge, Spinner, EmptyState, ProgressBar } from '../components/common'
import { initializeParameters, validateParameters, PARAMETER_TYPES } from '../types/plugins'

// Parameter input component
function ParameterInput({ param, value, onChange, error }) {
  const handleChange = (e) => {
    let newValue = e.target.value
    
    if (param.type === 'number' || param.type === 'range') {
      newValue = e.target.value === '' ? '' : Number(e.target.value)
    } else if (param.type === 'boolean') {
      newValue = e.target.checked
    }
    
    onChange(param.id, newValue)
  }

  const baseInputClass = `
    w-full px-4 py-2.5 bg-dark-900/50 border rounded-xl text-white 
    placeholder:text-dark-500 focus:outline-none focus:ring-2 
    transition-all duration-200
    ${error 
      ? 'border-red-500/50 focus:border-red-500/50 focus:ring-red-500/20' 
      : 'border-dark-800 focus:border-primary-500/50 focus:ring-primary-500/20'}
  `

  switch (param.type) {
    case 'select':
      return (
        <select
          value={value ?? ''}
          onChange={handleChange}
          className={baseInputClass}
        >
          <option value="">Select an option...</option>
          {param.options?.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label || opt.value}
            </option>
          ))}
        </select>
      )

    case 'boolean':
      return (
        <label className="flex items-center gap-3 cursor-pointer group">
          <div className="relative">
            <input
              type="checkbox"
              checked={value === true}
              onChange={handleChange}
              className="sr-only peer"
            />
            <div className="w-11 h-6 bg-dark-800 rounded-full peer-checked:bg-primary-500 transition-colors" />
            <div className="absolute left-1 top-1 w-4 h-4 bg-white rounded-full transition-transform peer-checked:translate-x-5" />
          </div>
          <span className="text-dark-300 group-hover:text-white transition-colors">
            {param.description}
          </span>
        </label>
      )

    case 'range':
      return (
        <div className="space-y-2">
          <div className="flex items-center gap-4">
            <input
              type="range"
              value={value ?? param.min ?? 0}
              onChange={handleChange}
              min={param.min ?? 0}
              max={param.max ?? 100}
              step={param.step ?? 1}
              className="flex-1 h-2 bg-dark-800 rounded-full appearance-none cursor-pointer accent-primary-500"
            />
            <span className="w-16 text-right font-mono text-white">
              {value ?? param.min ?? 0}
            </span>
          </div>
          <div className="flex justify-between text-xs text-dark-500">
            <span>{param.min ?? 0}</span>
            <span>{param.max ?? 100}</span>
          </div>
        </div>
      )

    case 'number':
      return (
        <input
          type="number"
          value={value ?? ''}
          onChange={handleChange}
          placeholder={param.placeholder || param.description}
          min={param.min}
          max={param.max}
          step={param.step ?? 1}
          className={baseInputClass}
        />
      )

    default:
      return (
        <input
          type="text"
          value={value ?? ''}
          onChange={handleChange}
          placeholder={param.placeholder || param.description}
          className={baseInputClass}
        />
      )
  }
}

// Result renderer component
function ResultRenderer({ result, pluginId }) {
  const [copied, setCopied] = useState(false)

  const copyToClipboard = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(JSON.stringify(result.data, null, 2))
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      console.error('Failed to copy')
    }
  }, [result.data])

  const downloadResult = useCallback(() => {
    const blob = new Blob([JSON.stringify(result.data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${pluginId}-result-${Date.now()}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }, [result.data, pluginId])

  if (!result) return null

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="space-y-4"
    >
      {/* Status indicator */}
      <div className="flex items-center justify-between">
        <div className={`flex items-center gap-2 ${result.success ? 'text-green-400' : 'text-red-400'}`}>
          {result.success ? (
            <CheckCircle className="w-5 h-5" />
          ) : (
            <AlertCircle className="w-5 h-5" />
          )}
          <span className="font-medium">
            {result.success ? 'Execution Successful' : 'Execution Failed'}
          </span>
        </div>
        
        <div className="flex items-center gap-2 text-xs text-dark-400">
          <Clock className="w-4 h-4" />
          {result.timestamp?.toLocaleTimeString()}
        </div>
      </div>

      {/* Error message */}
      {result.error && (
        <div className="p-4 bg-red-500/20 border border-red-500/30 rounded-xl text-red-400">
          {result.error}
        </div>
      )}

      {/* Result data */}
      {result.data && (
        <div className="relative">
          {/* Action buttons */}
          <div className="absolute top-3 right-3 flex gap-2">
            <button
              onClick={copyToClipboard}
              className="p-2 rounded-lg bg-dark-800/50 hover:bg-dark-800 text-dark-400 hover:text-white transition-colors"
              title="Copy to clipboard"
            >
              {copied ? <CheckCircle className="w-4 h-4 text-green-400" /> : <Copy className="w-4 h-4" />}
            </button>
            <button
              onClick={downloadResult}
              className="p-2 rounded-lg bg-dark-800/50 hover:bg-dark-800 text-dark-400 hover:text-white transition-colors"
              title="Download JSON"
            >
              <Download className="w-4 h-4" />
            </button>
          </div>

          {/* JSON display */}
          <pre className="bg-dark-900/50 p-4 pr-24 rounded-xl overflow-auto text-sm text-dark-200 font-mono max-h-96">
            {JSON.stringify(result.data, null, 2)}
          </pre>
        </div>
      )}
    </motion.div>
  )
}

export default function PluginPage() {
  const { id } = useParams()
  const { plugin, loading, error, running, result, run, clear } = usePlugin(id)
  const [params, setParams] = useState({})
  const [validationErrors, setValidationErrors] = useState({})
  const [iterateMode, setIterateMode] = useState(false)

  // Initialize parameters when plugin loads
  useEffect(() => {
    if (plugin?.Parameters) {
      setParams(initializeParameters(plugin.Parameters))
    }
  }, [plugin])

  const handleParamChange = useCallback((paramId, value) => {
    setParams(prev => ({ ...prev, [paramId]: value }))
    // Clear validation error when user changes value
    if (validationErrors[paramId]) {
      setValidationErrors(prev => {
        const next = { ...prev }
        delete next[paramId]
        return next
      })
    }
  }, [validationErrors])

  const handleRun = useCallback(async () => {
    // Validate parameters
    if (plugin?.Parameters) {
      const errors = validateParameters(plugin.Parameters, params)
      if (Object.keys(errors).length > 0) {
        setValidationErrors(errors)
        return
      }
    }

    try {
      await run(params)
    } catch {
      // Error is handled by the context
    }
  }, [plugin, params, run])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner size="lg" className="text-primary-400" />
      </div>
    )
  }

  if (error || !plugin) {
    return (
      <EmptyState
        icon={AlertCircle}
        title="Plugin Not Found"
        description={`The plugin "${id}" could not be found or failed to load.`}
        action={
          <Link to="/" className="btn-primary">
            Back to Dashboard
          </Link>
        }
      />
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link 
          to="/" 
          className="p-2 rounded-lg hover:bg-dark-800/50 transition-colors"
        >
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white">{plugin.Name}</h1>
            {plugin.Version && (
              <Badge variant="default" size="sm">v{plugin.Version}</Badge>
            )}
          </div>
          <p className="text-dark-400 mt-1">{plugin.Description}</p>
        </div>
        {plugin.Author && (
          <div className="text-right">
            <p className="text-xs text-dark-500">Author</p>
            <p className="text-sm text-dark-300">{plugin.Author}</p>
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Parameters Panel */}
        <div className="lg:col-span-1">
          <Card>
            <CardHeader
              icon={Settings}
              title="Parameters"
              iconColor="text-primary-400"
              iconBg="bg-primary-500/20"
            />

            <div className="space-y-5">
              {plugin.Parameters && plugin.Parameters.length > 0 ? (
                plugin.Parameters.map((param) => (
                  <div key={param.id || param.Name}>
                    <label className="block text-sm font-medium text-dark-300 mb-2">
                      {param.name || param.Name}
                      {param.required && <span className="text-red-400 ml-1">*</span>}
                      {param.canIterate && (
                        <Badge variant="cyan" size="sm" className="ml-2">Iterable</Badge>
                      )}
                    </label>
                    
                    <ParameterInput
                      param={param}
                      value={params[param.id || param.Name]}
                      onChange={handleParamChange}
                      error={validationErrors[param.id || param.Name]}
                    />
                    
                    {validationErrors[param.id || param.Name] && (
                      <p className="text-xs text-red-400 mt-1">
                        {validationErrors[param.id || param.Name]}
                      </p>
                    )}
                    
                    {param.description && param.type !== 'boolean' && (
                      <p className="text-xs text-dark-500 mt-1">{param.description}</p>
                    )}
                  </div>
                ))
              ) : (
                <p className="text-dark-400 text-sm">No parameters required</p>
              )}
            </div>

            {/* Iteration toggle */}
            {plugin.Parameters?.some(p => p.canIterate) && (
              <div className="mt-6 pt-6 border-t border-dark-800/50">
                <label className="flex items-center gap-3 cursor-pointer">
                  <div className="relative">
                    <input
                      type="checkbox"
                      checked={iterateMode}
                      onChange={(e) => setIterateMode(e.target.checked)}
                      className="sr-only peer"
                    />
                    <div className="w-11 h-6 bg-dark-800 rounded-full peer-checked:bg-cyan-500 transition-colors" />
                    <div className="absolute left-1 top-1 w-4 h-4 bg-white rounded-full transition-transform peer-checked:translate-x-5" />
                  </div>
                  <div>
                    <span className="text-white font-medium">Continuous Mode</span>
                    <p className="text-xs text-dark-400">Run plugin repeatedly</p>
                  </div>
                </label>
              </div>
            )}

            {/* Action buttons */}
            <div className="flex gap-3 mt-6">
              <Button
                onClick={handleRun}
                disabled={running}
                loading={running}
                icon={running ? undefined : (iterateMode ? Repeat : Play)}
                className="flex-1"
              >
                {running ? 'Running...' : (iterateMode ? 'Start Continuous' : 'Run Plugin')}
              </Button>
              
              {result && (
                <Button
                  variant="secondary"
                  onClick={clear}
                >
                  Clear
                </Button>
              )}
            </div>
          </Card>
        </div>

        {/* Results Panel */}
        <div className="lg:col-span-2">
          <Card className="min-h-[400px]">
            <CardHeader
              icon={running ? RefreshCw : CheckCircle}
              title="Results"
              iconColor={running ? 'text-cyan-400' : 'text-green-400'}
              iconBg={running ? 'bg-cyan-500/20' : 'bg-green-500/20'}
              badge={
                running && (
                  <Badge variant="cyan">
                    <Spinner size="sm" className="mr-1" />
                    Running
                  </Badge>
                )
              }
            />

            {running && (
              <div className="mb-4">
                <ProgressBar value={50} color="primary" />
                <p className="text-xs text-dark-400 mt-2 text-center">
                  Executing plugin...
                </p>
              </div>
            )}

            {result ? (
              <ResultRenderer result={result} pluginId={id} />
            ) : !running && (
              <EmptyState
                icon={Play}
                title="No Results Yet"
                description="Configure the parameters and run the plugin to see results"
              />
            )}
          </Card>
        </div>
      </div>
    </div>
  )
}
