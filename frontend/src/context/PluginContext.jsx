import { createContext, useContext, useState, useEffect, useCallback, useReducer } from 'react'
import { pluginsApi, pluginManagerApi } from '../api'

// Normalize plugin data to handle both PascalCase and lowercase JSON keys
function normalizePlugin(plugin) {
  return {
    ID: plugin.ID || plugin.id,
    Name: plugin.Name || plugin.name,
    Description: plugin.Description || plugin.description,
    Version: plugin.Version || plugin.version,
    Author: plugin.Author || plugin.author,
    License: plugin.License || plugin.license,
    Icon: plugin.Icon || plugin.icon,
    Parameters: (plugin.Parameters || plugin.parameters || []).map(param => ({
      id: param.id || param.ID,
      name: param.name || param.Name,
      description: param.description || param.Description,
      type: param.type || param.Type,
      required: param.required || param.Required || false,
      default: param.default || param.Default,
      options: param.options || param.Options,
      min: param.min || param.Min,
      max: param.max || param.Max,
      step: param.step || param.Step,
      canIterate: param.canIterate || param.CanIterate || false,
    })),
  }
}

// Action types for plugin state
const PLUGIN_ACTIONS = {
  SET_PLUGINS: 'SET_PLUGINS',
  SET_LOADING: 'SET_LOADING',
  SET_ERROR: 'SET_ERROR',
  SET_RUNNING: 'SET_RUNNING',
  SET_RESULT: 'SET_RESULT',
  CLEAR_RESULT: 'CLEAR_RESULT',
  SET_INSTALLED: 'SET_INSTALLED',
  SET_AVAILABLE: 'SET_AVAILABLE',
  UPDATE_PLUGIN: 'UPDATE_PLUGIN',
}

// Initial state
const initialState = {
  plugins: [],
  installedPlugins: [],
  availablePlugins: [],
  loading: false,
  error: null,
  runningPlugins: {}, // { pluginId: { running: boolean, progress: number, message: string } }
  results: {}, // { pluginId: { data: any, timestamp: Date, success: boolean } }
}

// Reducer for managing plugin state
function pluginReducer(state, action) {
  switch (action.type) {
    case PLUGIN_ACTIONS.SET_PLUGINS:
      return { ...state, plugins: action.payload, loading: false }
    
    case PLUGIN_ACTIONS.SET_LOADING:
      return { ...state, loading: action.payload }
    
    case PLUGIN_ACTIONS.SET_ERROR:
      return { ...state, error: action.payload, loading: false }
    
    case PLUGIN_ACTIONS.SET_RUNNING:
      return {
        ...state,
        runningPlugins: {
          ...state.runningPlugins,
          [action.payload.pluginId]: {
            running: action.payload.running,
            progress: action.payload.progress || 0,
            message: action.payload.message || '',
          },
        },
      }
    
    case PLUGIN_ACTIONS.SET_RESULT:
      return {
        ...state,
        results: {
          ...state.results,
          [action.payload.pluginId]: {
            data: action.payload.data,
            timestamp: new Date(),
            success: action.payload.success,
            error: action.payload.error,
          },
        },
        runningPlugins: {
          ...state.runningPlugins,
          [action.payload.pluginId]: { running: false, progress: 100, message: '' },
        },
      }
    
    case PLUGIN_ACTIONS.CLEAR_RESULT:
      const { [action.payload]: removed, ...remainingResults } = state.results
      return { ...state, results: remainingResults }
    
    case PLUGIN_ACTIONS.SET_INSTALLED:
      return { ...state, installedPlugins: action.payload }
    
    case PLUGIN_ACTIONS.SET_AVAILABLE:
      return { ...state, availablePlugins: action.payload }
    
    case PLUGIN_ACTIONS.UPDATE_PLUGIN:
      return {
        ...state,
        plugins: state.plugins.map(p => 
          p.ID === action.payload.id ? { ...p, ...action.payload.updates } : p
        ),
      }
    
    default:
      return state
  }
}

// Create context
const PluginContext = createContext(null)

// Plugin provider component
export function PluginProvider({ children }) {
  const [state, dispatch] = useReducer(pluginReducer, initialState)

  // Load all plugins on mount
  useEffect(() => {
    loadPlugins()
  }, [])

  // Load plugins from API
  const loadPlugins = useCallback(async () => {
    dispatch({ type: PLUGIN_ACTIONS.SET_LOADING, payload: true })
    dispatch({ type: PLUGIN_ACTIONS.SET_ERROR, payload: null })
    
    try {
      const response = await pluginsApi.getAll()
      const rawPlugins = response.data || []
      const normalizedPlugins = rawPlugins.map(normalizePlugin)
      dispatch({ type: PLUGIN_ACTIONS.SET_PLUGINS, payload: normalizedPlugins })
    } catch (err) {
      console.error('Failed to load plugins:', err)
      dispatch({ type: PLUGIN_ACTIONS.SET_ERROR, payload: 'Failed to load plugins' })
    }
  }, [])

  // Load installed and available plugins
  const loadPluginStore = useCallback(async () => {
    dispatch({ type: PLUGIN_ACTIONS.SET_LOADING, payload: true })
    dispatch({ type: PLUGIN_ACTIONS.SET_ERROR, payload: null })
    
    try {
      const [installed, available] = await Promise.all([
        pluginManagerApi.listInstalled(),
        pluginManagerApi.listAvailable(),
      ])
      
      dispatch({ type: PLUGIN_ACTIONS.SET_INSTALLED, payload: installed.data || [] })
      dispatch({ type: PLUGIN_ACTIONS.SET_AVAILABLE, payload: available.data || [] })
      dispatch({ type: PLUGIN_ACTIONS.SET_LOADING, payload: false })
    } catch (err) {
      console.error('Failed to load plugin store:', err)
      dispatch({ type: PLUGIN_ACTIONS.SET_ERROR, payload: 'Failed to load plugin store' })
    }
  }, [])

  // Get a single plugin by ID
  const getPlugin = useCallback(async (id) => {
    try {
      const response = await pluginsApi.getById(id)
      return normalizePlugin(response.data)
    } catch (err) {
      console.error(`Failed to get plugin ${id}:`, err)
      throw err
    }
  }, [])

  // Run a plugin
  const runPlugin = useCallback(async (id, params = {}, config = {}) => {
    dispatch({
      type: PLUGIN_ACTIONS.SET_RUNNING,
      payload: { pluginId: id, running: true, progress: 0, message: 'Starting...' },
    })

    try {
      const response = await pluginsApi.run(id, params)
      
      dispatch({
        type: PLUGIN_ACTIONS.SET_RESULT,
        payload: {
          pluginId: id,
          data: response.data,
          success: true,
        },
      })

      return response.data
    } catch (err) {
      const errorMessage = err.response?.data?.error || err.message || 'Plugin execution failed'
      
      dispatch({
        type: PLUGIN_ACTIONS.SET_RESULT,
        payload: {
          pluginId: id,
          data: null,
          success: false,
          error: errorMessage,
        },
      })

      throw new Error(errorMessage)
    }
  }, [])

  // Clear plugin result
  const clearResult = useCallback((pluginId) => {
    dispatch({ type: PLUGIN_ACTIONS.CLEAR_RESULT, payload: pluginId })
  }, [])

  // Install a plugin
  const installPlugin = useCallback(async (repository) => {
    try {
      await pluginManagerApi.install(repository)
      await loadPlugins()
      await loadPluginStore()
      return true
    } catch (err) {
      console.error('Failed to install plugin:', err)
      throw err
    }
  }, [loadPlugins, loadPluginStore])

  // Uninstall a plugin
  const uninstallPlugin = useCallback(async (id) => {
    try {
      await pluginManagerApi.uninstall(id)
      await loadPlugins()
      await loadPluginStore()
      return true
    } catch (err) {
      console.error('Failed to uninstall plugin:', err)
      throw err
    }
  }, [loadPlugins, loadPluginStore])

  // Update a plugin
  const updatePlugin = useCallback(async (id) => {
    try {
      await pluginManagerApi.update(id)
      await loadPlugins()
      await loadPluginStore()
      return true
    } catch (err) {
      console.error('Failed to update plugin:', err)
      throw err
    }
  }, [loadPlugins, loadPluginStore])

  // Refresh plugin catalog
  const refreshCatalog = useCallback(async () => {
    try {
      await pluginManagerApi.refreshCatalog()
      await loadPluginStore()
      return true
    } catch (err) {
      console.error('Failed to refresh catalog:', err)
      throw err
    }
  }, [loadPluginStore])

  // Check if a plugin is running
  const isPluginRunning = useCallback((pluginId) => {
    return state.runningPlugins[pluginId]?.running || false
  }, [state.runningPlugins])

  // Get plugin result
  const getPluginResult = useCallback((pluginId) => {
    return state.results[pluginId] || null
  }, [state.results])

  // Get plugins by category
  const getPluginsByCategory = useCallback((category) => {
    const categoryMappings = {
      'network-analysis': ['network_quality', 'bandwidth_test', 'packet_capture', 'network_info', 'iperf3', 'tc_controller', 'network_latency_heatmap', 'subnet_calculator'],
      'network-discovery': ['device_discovery', 'port_scanner', 'wifi_scanner'],
      'connectivity': ['ping', 'traceroute', 'mtu_tester'],
      'performance': ['iperf3', 'iperf3_server', 'tc_controller'],
      'dns-tools': ['dns_lookup', 'dns_propagation', 'reverse_dns_lookup'],
      'security': ['ssl_checker'],
    }

    const pluginIds = categoryMappings[category] || []
    return state.plugins.filter(p => pluginIds.includes(p.ID))
  }, [state.plugins])

  const value = {
    // State
    plugins: state.plugins,
    installedPlugins: state.installedPlugins,
    availablePlugins: state.availablePlugins,
    loading: state.loading,
    error: state.error,
    runningPlugins: state.runningPlugins,
    results: state.results,

    // Actions
    loadPlugins,
    loadPluginStore,
    getPlugin,
    runPlugin,
    clearResult,
    installPlugin,
    uninstallPlugin,
    updatePlugin,
    refreshCatalog,

    // Helpers
    isPluginRunning,
    getPluginResult,
    getPluginsByCategory,
  }

  return (
    <PluginContext.Provider value={value}>
      {children}
    </PluginContext.Provider>
  )
}

// Custom hook to use plugin context
export function usePlugins() {
  const context = useContext(PluginContext)
  if (!context) {
    throw new Error('usePlugins must be used within a PluginProvider')
  }
  return context
}

// Hook for single plugin operations
export function usePlugin(pluginId) {
  const {
    getPlugin,
    runPlugin,
    clearResult,
    isPluginRunning,
    getPluginResult,
  } = usePlugins()

  const [plugin, setPlugin] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  useEffect(() => {
    let mounted = true

    async function load() {
      if (!pluginId) return
      
      setLoading(true)
      setError(null)
      
      try {
        const data = await getPlugin(pluginId)
        if (mounted) {
          setPlugin(data)
        }
      } catch (err) {
        if (mounted) {
          setError(err.message)
        }
      } finally {
        if (mounted) {
          setLoading(false)
        }
      }
    }

    load()

    return () => {
      mounted = false
    }
  }, [pluginId, getPlugin])

  const run = useCallback(async (params = {}) => {
    return runPlugin(pluginId, params)
  }, [pluginId, runPlugin])

  const clear = useCallback(() => {
    clearResult(pluginId)
  }, [pluginId, clearResult])

  return {
    plugin,
    loading,
    error,
    running: isPluginRunning(pluginId),
    result: getPluginResult(pluginId),
    run,
    clear,
  }
}

export default PluginContext
