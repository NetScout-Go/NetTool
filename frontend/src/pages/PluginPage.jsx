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
  StopCircle,
  Wifi,
  Network
} from 'lucide-react'
import { usePlugin } from '../context/PluginContext'
import { Card, CardHeader, Button, Badge, Spinner, EmptyState, ProgressBar } from '../components/common'
import { initializeParameters, validateParameters, PARAMETER_TYPES } from '../types/plugins'
import { interfacesApi } from '../api'

// Check if parameter should use interface selection
function isInterfaceParam(param) {
  const name = (param.name || param.Name || param.id || '').toLowerCase()
  const desc = (param.description || '').toLowerCase()
  
  // Check for interface-related keywords
  const interfaceKeywords = ['interface', 'iface', 'device', 'adapter', 'nic']
  const wifiKeywords = ['wifi', 'wireless', 'wlan', 'wi-fi']
  const ethKeywords = ['ethernet', 'eth', 'lan', 'wired']
  
  const isInterface = interfaceKeywords.some(k => name.includes(k) || desc.includes(k))
  const isWifi = wifiKeywords.some(k => name.includes(k) || desc.includes(k))
  const isEth = ethKeywords.some(k => name.includes(k) || desc.includes(k))
  
  if (isWifi) return 'wifi'
  if (isEth) return 'ethernet'
  if (isInterface) return 'all'
  return null
}

// Interface selector component with auto-detection
function InterfaceSelector({ param, value, onChange, error, interfaceType }) {
  const [interfaces, setInterfaces] = useState(null)
  const [loading, setLoading] = useState(true)
  const [fetchError, setFetchError] = useState(null)

  useEffect(() => {
    const fetchInterfaces = async () => {
      try {
        setLoading(true)
        const response = await interfacesApi.getAll()
        setInterfaces(response.data)
        
        // Auto-select primary interface if no value set
        if (!value && response.data) {
          let autoSelect = null
          if (interfaceType === 'wifi' && response.data.primaryWifi) {
            autoSelect = response.data.primaryWifi.name
          } else if (interfaceType === 'ethernet' && response.data.primaryEthernet) {
            autoSelect = response.data.primaryEthernet.name
          } else if (response.data.primary) {
            autoSelect = response.data.primary.name
          }
          if (autoSelect) {
            onChange(param.id || param.Name, autoSelect)
          }
        }
      } catch (err) {
        console.error('Failed to fetch interfaces:', err)
        setFetchError(err.message)
      } finally {
        setLoading(false)
      }
    }
    fetchInterfaces()
  }, [interfaceType])

  if (loading) {
    return (
      <div className="flex items-center gap-2 px-4 py-2.5 bg-dark-900/50 border border-dark-800 rounded-xl">
        <Spinner size="sm" className="text-primary-400" />
        <span className="text-dark-400">Detecting interfaces...</span>
      </div>
    )
  }

  if (fetchError) {
    return (
      <input
        type="text"
        value={value ?? ''}
        onChange={(e) => onChange(param.id || param.Name, e.target.value)}
        placeholder="Enter interface name (e.g., wlan0, eth0)"
        className={`
          w-full px-4 py-2.5 bg-dark-900/50 border rounded-xl text-white 
          placeholder:text-dark-500 focus:outline-none focus:ring-2 
          transition-all duration-200 border-dark-800 focus:border-primary-500/50 focus:ring-primary-500/20
        `}
      />
    )
  }

  // Filter interfaces based on type
  let availableInterfaces = interfaces?.all || []
  if (interfaceType === 'wifi') {
    availableInterfaces = interfaces?.wifi || []
  } else if (interfaceType === 'ethernet') {
    availableInterfaces = interfaces?.ethernet || []
  } else {
    // For 'all', filter out loopback
    availableInterfaces = availableInterfaces.filter(i => i.type !== 'loopback')
  }

  const baseInputClass = `
    w-full px-4 py-2.5 bg-dark-900/50 border rounded-xl text-white 
    placeholder:text-dark-500 focus:outline-none focus:ring-2 
    transition-all duration-200
    ${error 
      ? 'border-red-500/50 focus:border-red-500/50 focus:ring-red-500/20' 
      : 'border-dark-800 focus:border-primary-500/50 focus:ring-primary-500/20'}
  `

  return (
    <div className="space-y-2">
      <select
        value={value ?? ''}
        onChange={(e) => onChange(param.id || param.Name, e.target.value)}
        className={baseInputClass}
      >
        <option value="">Select interface...</option>
        {availableInterfaces.map((iface) => (
          <option key={iface.name} value={iface.name}>
            {iface.name} 
            {iface.ipv4 ? ` (${iface.ipv4})` : ''} 
            {iface.type === 'wifi' ? ' - WiFi' : ''}
            {iface.type === 'ethernet' ? ' - Ethernet' : ''}
            {!iface.isUp ? ' [Down]' : ''}
          </option>
        ))}
      </select>
      
      {/* Show interface details for selected */}
      {value && availableInterfaces.length > 0 && (
        <div className="flex flex-wrap gap-2 text-xs">
          {(() => {
            const selected = availableInterfaces.find(i => i.name === value)
            if (!selected) return null
            return (
              <>
                {selected.type && (
                  <Badge variant={selected.type === 'wifi' ? 'cyan' : 'green'} size="sm">
                    {selected.type === 'wifi' ? <Wifi className="w-3 h-3 mr-1" /> : <Network className="w-3 h-3 mr-1" />}
                    {selected.type}
                  </Badge>
                )}
                {selected.ssid && (
                  <Badge variant="purple" size="sm">📶 {selected.ssid}</Badge>
                )}
                {selected.speed && (
                  <Badge variant="default" size="sm">⚡ {selected.speed}</Badge>
                )}
                {selected.isUp && (
                  <Badge variant="green" size="sm">● Active</Badge>
                )}
              </>
            )
          })()}
        </div>
      )}
    </div>
  )
}

// Parameter input component
function ParameterInput({ param, value, onChange, error }) {
  // Check if this is an interface parameter and use InterfaceSelector
  const interfaceType = isInterfaceParam(param)
  if (interfaceType && param.type !== 'select') {
    return (
      <InterfaceSelector 
        param={param} 
        value={value} 
        onChange={onChange} 
        error={error}
        interfaceType={interfaceType}
      />
    )
  }

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

// Format bytes to human readable
function formatBytes(bytes) {
  if (bytes === 0 || bytes === undefined || bytes === null) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

// Get color class from color name
function getColorClass(color, type = 'text') {
  const colorMap = {
    primary: type === 'bg' ? 'bg-primary-500/20' : 'text-primary-400',
    cyan: type === 'bg' ? 'bg-cyan-500/20' : 'text-cyan-400',
    green: type === 'bg' ? 'bg-green-500/20' : 'text-green-400',
    orange: type === 'bg' ? 'bg-orange-500/20' : 'text-orange-400',
    purple: type === 'bg' ? 'bg-purple-500/20' : 'text-purple-400',
    red: type === 'bg' ? 'bg-red-500/20' : 'text-red-400',
    yellow: type === 'bg' ? 'bg-yellow-500/20' : 'text-yellow-400',
    blue: type === 'bg' ? 'bg-blue-500/20' : 'text-blue-400',
  }
  return colorMap[color] || (type === 'bg' ? 'bg-dark-700/50' : 'text-dark-300')
}

// Render a metric item
function MetricItem({ metric }) {
  const colorClass = metric.color ? getColorClass(metric.color) : 'text-white'
  return (
    <div className="flex items-center justify-between py-2 border-b border-dark-800/30 last:border-0">
      <span className="text-dark-400 text-sm">{metric.label}</span>
      <span className={`font-medium ${colorClass}`}>
        {metric.value !== undefined && metric.value !== null && metric.value !== '' 
          ? String(metric.value) 
          : '--'}
        {metric.unit && <span className="text-dark-500 ml-1">{metric.unit}</span>}
      </span>
    </div>
  )
}

// Render a display section based on type
function DisplaySection({ section }) {
  const bgClass = getColorClass(section.color, 'bg')
  const textClass = getColorClass(section.color)

  // Metrics display
  if (section.type === 'metrics' && section.metrics) {
    return (
      <div className="bg-dark-900/30 rounded-xl p-4 border border-dark-800/50">
        <div className="flex items-center gap-2 mb-4">
          <div className={`p-2 rounded-lg ${bgClass}`}>
            <span className={textClass}>●</span>
          </div>
          <h4 className="font-semibold text-white">{section.title}</h4>
        </div>
        <div className="space-y-1">
          {section.metrics.map((metric, idx) => (
            <MetricItem key={idx} metric={metric} />
          ))}
        </div>
        {section.extra?.progress !== undefined && (
          <div className="mt-3">
            <div className="h-2 bg-dark-800 rounded-full overflow-hidden">
              <div 
                className={`h-full transition-all ${
                  section.extra.progress > 90 ? 'bg-red-500' :
                  section.extra.progress > 70 ? 'bg-orange-500' :
                  'bg-green-500'
                }`}
                style={{ width: `${Math.min(section.extra.progress, 100)}%` }}
              />
            </div>
          </div>
        )}
      </div>
    )
  }

  // Table display
  if (section.type === 'table' && section.columns && section.data) {
    return (
      <div className="bg-dark-900/30 rounded-xl p-4 border border-dark-800/50 overflow-x-auto">
        <div className="flex items-center gap-2 mb-4">
          <div className={`p-2 rounded-lg ${bgClass}`}>
            <span className={textClass}>●</span>
          </div>
          <h4 className="font-semibold text-white">{section.title}</h4>
        </div>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-dark-700">
              {section.columns.map((col, idx) => (
                <th key={idx} className="text-left py-2 px-3 text-dark-400 font-medium">
                  {col.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {section.data.map((row, rowIdx) => (
              <tr key={rowIdx} className="border-b border-dark-800/30 hover:bg-dark-800/20">
                {section.columns.map((col, colIdx) => {
                  let value = row[col.key]
                  let cellClass = 'text-dark-200'
                  
                  // Format based on type
                  if (col.type === 'bytes' && typeof value === 'number') {
                    value = formatBytes(value)
                  } else if (col.type === 'status') {
                    const isUp = value === 'up' || value === 'active' || value === 'connected'
                    cellClass = isUp ? 'text-green-400' : 'text-red-400'
                    value = (
                      <span className="flex items-center gap-1">
                        <span className={`w-2 h-2 rounded-full ${isUp ? 'bg-green-400' : 'bg-red-400'}`} />
                        {value}
                      </span>
                    )
                  } else if (col.type === 'progress') {
                    const pct = typeof value === 'number' ? value : 0
                    const color = pct > 90 ? 'bg-red-500' : pct > 70 ? 'bg-orange-500' : 'bg-green-500'
                    value = (
                      <div className="flex items-center gap-2">
                        <div className="w-16 h-1.5 bg-dark-700 rounded-full overflow-hidden">
                          <div className={`h-full ${color}`} style={{ width: `${pct}%` }} />
                        </div>
                        <span className="text-xs">{pct.toFixed(1)}%</span>
                      </div>
                    )
                  }
                  
                  return (
                    <td key={colIdx} className={`py-2 px-3 ${cellClass}`}>
                      {value !== undefined && value !== null && value !== '' ? value : '--'}
                    </td>
                  )
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )
  }

  // List display
  if (section.type === 'list' && section.data) {
    return (
      <div className="bg-dark-900/30 rounded-xl p-4 border border-dark-800/50">
        <div className="flex items-center gap-2 mb-4">
          <div className={`p-2 rounded-lg ${bgClass}`}>
            <span className={textClass}>●</span>
          </div>
          <h4 className="font-semibold text-white">{section.title}</h4>
        </div>
        <ul className="space-y-2">
          {(Array.isArray(section.data) ? section.data : []).map((item, idx) => (
            <li key={idx} className="flex items-center gap-2 text-dark-200">
              <span className={`w-1.5 h-1.5 rounded-full ${getColorClass(section.color, 'bg').replace('/20', '')}`} />
              {typeof item === 'object' ? JSON.stringify(item) : String(item)}
            </li>
          ))}
        </ul>
      </div>
    )
  }

  return null
}

// Result renderer component
function ResultRenderer({ result, pluginId }) {
  const [copied, setCopied] = useState(false)
  const [showRaw, setShowRaw] = useState(false)

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

  // Check if result has formatted display data
  const displaySections = result.data?._display || result.data?.Display || []
  const hasFormattedDisplay = displaySections.length > 0
  const executionTime = result.data?.executionTime || result.data?.ExecutionTime
  const warnings = result.data?.warnings || result.data?.Warnings || []
  const depStatus = result.data?.dependencyStatus || result.data?.DependencyStatus || []

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
          {executionTime && (
            <span className="text-dark-500 text-sm ml-2">({executionTime})</span>
          )}
        </div>
        
        <div className="flex items-center gap-2">
          {hasFormattedDisplay && (
            <button
              onClick={() => setShowRaw(!showRaw)}
              className="px-3 py-1 text-xs rounded-lg bg-dark-800/50 hover:bg-dark-800 text-dark-400 hover:text-white transition-colors"
            >
              {showRaw ? 'Show Formatted' : 'Show Raw'}
            </button>
          )}
          <div className="flex items-center gap-2 text-xs text-dark-400">
            <Clock className="w-4 h-4" />
            {result.timestamp?.toLocaleTimeString()}
          </div>
        </div>
      </div>

      {/* Warnings */}
      {warnings.length > 0 && (
        <div className="p-3 bg-yellow-500/10 border border-yellow-500/20 rounded-xl">
          <div className="flex items-center gap-2 text-yellow-400 mb-2">
            <AlertCircle className="w-4 h-4" />
            <span className="font-medium text-sm">Warnings</span>
          </div>
          <ul className="space-y-1">
            {warnings.map((warning, idx) => (
              <li key={idx} className="text-sm text-yellow-300/80 flex items-center gap-2">
                <span className="w-1 h-1 rounded-full bg-yellow-400" />
                {warning}
              </li>
            ))}
          </ul>
        </div>
      )}

      {/* Dependency Status */}
      {depStatus.length > 0 && depStatus.some(d => !d.installed) && (
        <div className="p-3 bg-dark-800/50 border border-dark-700 rounded-xl">
          <div className="flex items-center gap-2 text-dark-300 mb-2">
            <Settings className="w-4 h-4" />
            <span className="font-medium text-sm">Dependencies</span>
          </div>
          <div className="flex flex-wrap gap-2">
            {depStatus.map((dep, idx) => (
              <span 
                key={idx} 
                className={`px-2 py-1 rounded text-xs ${
                  dep.installed 
                    ? 'bg-green-500/20 text-green-400' 
                    : 'bg-red-500/20 text-red-400'
                }`}
                title={dep.installed ? dep.version : dep.installCmd}
              >
                {dep.installed ? '✓' : '✗'} {dep.name}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Error message */}
      {result.error && (
        <div className="p-4 bg-red-500/20 border border-red-500/30 rounded-xl">
          <div className="flex items-center gap-2 text-red-400 mb-1">
            <AlertCircle className="w-4 h-4" />
            <span className="font-medium">{result.data?.errorCode || 'Error'}</span>
          </div>
          <p className="text-red-400">{result.error}</p>
          {result.data?.errorDetails && (
            <p className="text-red-400/70 text-sm mt-2">{result.data.errorDetails}</p>
          )}
        </div>
      )}

      {/* Formatted display sections */}
      {hasFormattedDisplay && !showRaw && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {displaySections.map((section, idx) => (
            <DisplaySection key={idx} section={section} />
          ))}
        </div>
      )}

      {/* Raw JSON display */}
      {result.data && (!hasFormattedDisplay || showRaw) && (
        <div className="relative">
          {/* Action buttons */}
          <div className="absolute top-3 right-3 flex gap-2 z-10">
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
