import { useState, useEffect, useMemo } from 'react'
import { motion } from 'framer-motion'
import { 
  Settings, 
  Download, 
  RefreshCw, 
  Trash2, 
  Check, 
  X,
  ExternalLink,
  Package,
  Clock,
  AlertCircle,
  Github,
  Tag,
  User,
  Database,
  CheckSquare,
  Square,
  Layers
} from 'lucide-react'
import { pluginManagerApi } from '../api'

export default function PluginManager() {
  const [installedPlugins, setInstalledPlugins] = useState([])
  const [availablePlugins, setAvailablePlugins] = useState([])
  const [lastUpdated, setLastUpdated] = useState(null)
  const [fromCache, setFromCache] = useState(false)
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('available')
  const [error, setError] = useState(null)
  const [actionLoading, setActionLoading] = useState({})
  const [selectedPlugins, setSelectedPlugins] = useState(new Set())
  const [isMultiInstalling, setIsMultiInstalling] = useState(false)

  // Helper to get property with fallback for both casing styles
  const getPluginProp = (plugin, prop) => {
    // Try lowercase first (API returns), then PascalCase
    return plugin[prop] || plugin[prop.charAt(0).toUpperCase() + prop.slice(1)]
  }

  // Get plugin repository
  const getPluginRepository = (plugin) => getPluginProp(plugin, 'repository')
  
  // Check if plugin is installed
  const isInstalled = (plugin) => plugin.installed || plugin.Installed

  useEffect(() => {
    loadPlugins()
  }, [])

  const loadPlugins = async () => {
    setLoading(true)
    setError(null)
    try {
      const [installed, available] = await Promise.all([
        pluginManagerApi.listInstalled(),
        pluginManagerApi.listAvailable()
      ])
      setInstalledPlugins(installed.data || [])
      
      // Handle new response format with metadata
      const availableData = available.data
      if (availableData && 'plugins' in availableData) {
        setAvailablePlugins(availableData.plugins || [])
        setLastUpdated(availableData.lastUpdated ? new Date(availableData.lastUpdated * 1000) : null)
        setFromCache(availableData.fromCache || false)
      } else if (Array.isArray(availableData)) {
        // Fallback for old format (array directly)
        setAvailablePlugins(availableData)
      } else {
        setAvailablePlugins([])
      }
    } catch (err) {
      setError('Failed to load plugins: ' + (err.message || 'Unknown error'))
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleInstall = async (repository) => {
    setActionLoading(prev => ({ ...prev, [repository]: true }))
    try {
      await pluginManagerApi.install(repository)
      await loadPlugins()
      // Remove from selection after successful install
      setSelectedPlugins(prev => {
        const next = new Set(prev)
        next.delete(repository)
        return next
      })
    } catch (err) {
      console.error('Failed to install plugin:', err)
      setError('Failed to install plugin: ' + (err.message || 'Unknown error'))
    } finally {
      setActionLoading(prev => ({ ...prev, [repository]: false }))
    }
  }

  // Multi-install handler
  const handleInstallSelected = async () => {
    if (selectedPlugins.size === 0) return
    
    setIsMultiInstalling(true)
    const repositories = Array.from(selectedPlugins)
    const errors = []
    
    for (const repository of repositories) {
      setActionLoading(prev => ({ ...prev, [repository]: true }))
      try {
        await pluginManagerApi.install(repository)
      } catch (err) {
        console.error(`Failed to install ${repository}:`, err)
        errors.push(`${repository}: ${err.message || 'Unknown error'}`)
      } finally {
        setActionLoading(prev => ({ ...prev, [repository]: false }))
      }
    }
    
    // Reload plugins and clear selection
    await loadPlugins()
    setSelectedPlugins(new Set())
    setIsMultiInstalling(false)
    
    if (errors.length > 0) {
      setError(`Some plugins failed to install:\n${errors.join('\n')}`)
    }
  }

  // Toggle plugin selection
  const togglePluginSelection = (repository) => {
    setSelectedPlugins(prev => {
      const next = new Set(prev)
      if (next.has(repository)) {
        next.delete(repository)
      } else {
        next.add(repository)
      }
      return next
    })
  }

  // Get available (not installed) plugins
  const availableNotInstalled = useMemo(() => 
    availablePlugins.filter(p => !isInstalled(p)),
    [availablePlugins]
  )

  // Select/deselect all available plugins
  const toggleSelectAll = () => {
    if (selectedPlugins.size === availableNotInstalled.length) {
      setSelectedPlugins(new Set())
    } else {
      const allRepos = availableNotInstalled
        .map(p => getPluginRepository(p))
        .filter(Boolean)
      setSelectedPlugins(new Set(allRepos))
    }
  }

  const allSelected = availableNotInstalled.length > 0 && 
    selectedPlugins.size === availableNotInstalled.length

  const handleUpdate = async (id) => {
    setActionLoading(prev => ({ ...prev, [id]: true }))
    try {
      await pluginManagerApi.update(id)
      await loadPlugins()
    } catch (err) {
      console.error('Failed to update plugin:', err)
    } finally {
      setActionLoading(prev => ({ ...prev, [id]: false }))
    }
  }

  const handleUninstall = async (id) => {
    if (!confirm('Are you sure you want to uninstall this plugin?')) return
    
    setActionLoading(prev => ({ ...prev, [id]: true }))
    try {
      await pluginManagerApi.uninstall(id)
      await loadPlugins()
    } catch (err) {
      console.error('Failed to uninstall plugin:', err)
    } finally {
      setActionLoading(prev => ({ ...prev, [id]: false }))
    }
  }

  const handleRefreshCatalog = async () => {
    setActionLoading(prev => ({ ...prev, refresh: true }))
    setError(null)
    try {
      const response = await pluginManagerApi.refreshCatalog()
      if (response.data?.plugins) {
        setAvailablePlugins(response.data.plugins || [])
        setLastUpdated(response.data.lastUpdated ? new Date(response.data.lastUpdated * 1000) : new Date())
        setFromCache(false)
      } else {
        await loadPlugins()
      }
    } catch (err) {
      console.error('Failed to refresh catalog:', err)
      setError('Failed to refresh catalog: ' + (err.message || 'Unknown error'))
    } finally {
      setActionLoading(prev => ({ ...prev, refresh: false }))
    }
  }

  const formatLastUpdated = () => {
    if (!lastUpdated) return null
    const now = new Date()
    const diff = now - lastUpdated
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(diff / 3600000)
    const days = Math.floor(diff / 86400000)
    
    if (minutes < 1) return 'just now'
    if (minutes < 60) return `${minutes} minute${minutes !== 1 ? 's' : ''} ago`
    if (hours < 24) return `${hours} hour${hours !== 1 ? 's' : ''} ago`
    if (days < 7) return `${days} day${days !== 1 ? 's' : ''} ago`
    return lastUpdated.toLocaleDateString()
  }

  // Format plugin name from ID
  const formatPluginName = (plugin) => {
    const name = getPluginProp(plugin, 'name')
    if (name && name !== getPluginProp(plugin, 'id')) return name
    const id = getPluginProp(plugin, 'id') || 'Unknown Plugin'
    return id
      .replace(/^Plugin_/, '')
      .replace(/_/g, ' ')
      .replace(/\b\w/g, c => c.toUpperCase())
  }

  const getPluginId = (plugin) => getPluginProp(plugin, 'id')
  const getPluginDescription = (plugin) => getPluginProp(plugin, 'description') || 'No description available'
  const getPluginVersion = (plugin) => getPluginProp(plugin, 'version') || '1.0.0'
  const getPluginAuthor = (plugin) => getPluginProp(plugin, 'author')
  const hasUpdate = (plugin) => plugin.updateAvailable || plugin.UpdateAvailable

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Plugin Manager</h1>
          <div className="flex items-center gap-3 mt-1">
            <p className="text-dark-400">Install and manage network analysis plugins</p>
            {lastUpdated && (
              <div className="flex items-center gap-1.5 text-xs text-dark-500">
                <span className="text-dark-600">•</span>
                {fromCache && <Database className="w-3 h-3" />}
                <Clock className="w-3 h-3" />
                <span>Updated {formatLastUpdated()}</span>
              </div>
            )}
          </div>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={handleRefreshCatalog}
            disabled={actionLoading.refresh}
            className="btn-secondary flex items-center gap-2"
          >
            <RefreshCw className={`w-4 h-4 ${actionLoading.refresh ? 'animate-spin' : ''}`} />
            Refresh Catalog
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 p-1 bg-dark-900/50 rounded-xl w-fit">
        <button
          onClick={() => setActiveTab('available')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
            activeTab === 'available'
              ? 'bg-primary-500/20 text-primary-400'
              : 'text-dark-400 hover:text-white'
          }`}
        >
          Available ({availableNotInstalled.length})
        </button>
        <button
          onClick={() => setActiveTab('installed')}
          className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
            activeTab === 'installed'
              ? 'bg-primary-500/20 text-primary-400'
              : 'text-dark-400 hover:text-white'
          }`}
        >
          Installed ({installedPlugins.length})
        </button>
      </div>

      {/* Multi-select toolbar - only show when on available tab */}
      {activeTab === 'available' && availableNotInstalled.length > 0 && (
        <div className="flex items-center justify-between p-3 bg-dark-900/50 rounded-xl border border-dark-700/50">
          <div className="flex items-center gap-3">
            <button
              onClick={toggleSelectAll}
              className="flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm font-medium transition-all hover:bg-dark-700/50 text-dark-300 hover:text-white"
            >
              {allSelected ? (
                <CheckSquare className="w-4 h-4 text-primary-400" />
              ) : (
                <Square className="w-4 h-4" />
              )}
              {allSelected ? 'Deselect All' : 'Select All'}
            </button>
            
            {selectedPlugins.size > 0 && (
              <span className="text-sm text-dark-400">
                {selectedPlugins.size} plugin{selectedPlugins.size !== 1 ? 's' : ''} selected
              </span>
            )}
          </div>
          
          {selectedPlugins.size > 0 && (
            <button
              onClick={handleInstallSelected}
              disabled={isMultiInstalling}
              className="btn-primary flex items-center gap-2"
            >
              {isMultiInstalling ? (
                <>
                  <RefreshCw className="w-4 h-4 animate-spin" />
                  Installing...
                </>
              ) : (
                <>
                  <Layers className="w-4 h-4" />
                  Install Selected ({selectedPlugins.size})
                </>
              )}
            </button>
          )}
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="flex items-center gap-3 p-4 bg-red-500/20 border border-red-500/30 rounded-xl text-red-400">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <span className="flex-1 whitespace-pre-wrap">{error}</span>
          <button onClick={() => setError(null)} className="text-red-300 hover:text-white">
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Loading */}
      {loading ? (
        <div className="flex flex-col items-center justify-center py-12">
          <RefreshCw className="w-8 h-8 text-primary-400 animate-spin mb-4" />
          <p className="text-dark-400">Loading plugins from GitHub...</p>
        </div>
      ) : (
        <>
          {/* Installed Plugins */}
          {activeTab === 'installed' && (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {installedPlugins.length > 0 ? (
                installedPlugins.map((plugin, index) => {
                  const pluginId = getPluginId(plugin)
                  return (
                    <motion.div
                      key={pluginId || index}
                      initial={{ opacity: 0, y: 20 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: index * 0.05 }}
                      className="glass-card p-5"
                    >
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex items-center gap-3">
                          <div className="w-10 h-10 rounded-lg bg-primary-500/20 flex items-center justify-center">
                            <Package className="w-5 h-5 text-primary-400" />
                          </div>
                          <div>
                            <h3 className="font-semibold text-white">{formatPluginName(plugin)}</h3>
                            <div className="flex items-center gap-2 mt-0.5">
                              <Tag className="w-3 h-3 text-dark-500" />
                              <span className="text-xs text-dark-400">v{getPluginVersion(plugin)}</span>
                            </div>
                          </div>
                        </div>
                        {hasUpdate(plugin) && (
                          <span className="status-badge warning text-xs">
                            Update
                          </span>
                        )}
                      </div>
                      
                      <p className="text-sm text-dark-300 mb-4 line-clamp-2">
                        {getPluginDescription(plugin)}
                      </p>

                      <div className="flex items-center gap-2">
                        {hasUpdate(plugin) && (
                          <button
                            onClick={() => handleUpdate(pluginId)}
                            disabled={actionLoading[pluginId]}
                            className="btn-primary text-sm flex-1"
                          >
                            {actionLoading[pluginId] ? (
                              <RefreshCw className="w-4 h-4 animate-spin" />
                            ) : (
                              <>
                                <Download className="w-4 h-4 inline mr-1" />
                                Update
                              </>
                            )}
                          </button>
                        )}
                        <button
                          onClick={() => handleUninstall(pluginId)}
                          disabled={actionLoading[pluginId]}
                          className="p-2 rounded-lg bg-red-500/20 text-red-400 hover:bg-red-500/30 transition-colors"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </motion.div>
                  )
                })
              ) : (
                <div className="col-span-full text-center py-12 text-dark-400">
                  <Package className="w-12 h-12 mx-auto mb-4 opacity-50" />
                  <p>No plugins installed</p>
                  <p className="text-sm mt-1">Browse available plugins to get started</p>
                </div>
              )}
            </div>
          )}

          {/* Available Plugins */}
          {activeTab === 'available' && (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {availableNotInstalled.length > 0 ? (
                availableNotInstalled.map((plugin, index) => {
                  const pluginId = getPluginId(plugin)
                  const repository = getPluginRepository(plugin)
                  const author = getPluginAuthor(plugin)
                  const category = getPluginProp(plugin, 'category')
                  const tags = plugin.tags || plugin.Tags || []
                  const isSelected = repository && selectedPlugins.has(repository)
                  
                  return (
                    <motion.div
                      key={repository || pluginId || index}
                      initial={{ opacity: 0, y: 20 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ delay: index * 0.05 }}
                      className={`glass-card p-5 transition-all cursor-pointer ${
                        isSelected 
                          ? 'border-primary-500/50 bg-primary-500/5' 
                          : 'hover:border-primary-500/30'
                      }`}
                      onClick={() => repository && togglePluginSelection(repository)}
                    >
                      <div className="flex items-start justify-between mb-3">
                        <div className="flex items-center gap-3">
                          {/* Selection checkbox */}
                          <div 
                            className={`w-10 h-10 rounded-lg flex items-center justify-center transition-colors ${
                              isSelected 
                                ? 'bg-primary-500/20' 
                                : 'bg-dark-700/50'
                            }`}
                          >
                            {isSelected ? (
                              <CheckSquare className="w-5 h-5 text-primary-400" />
                            ) : (
                              <Package className="w-5 h-5 text-dark-300" />
                            )}
                          </div>
                          <div>
                            <h3 className="font-semibold text-white">{formatPluginName(plugin)}</h3>
                            <div className="flex items-center gap-3 mt-0.5">
                              <div className="flex items-center gap-1">
                                <Tag className="w-3 h-3 text-dark-500" />
                                <span className="text-xs text-dark-400">v{getPluginVersion(plugin)}</span>
                              </div>
                              {author && (
                                <div className="flex items-center gap-1">
                                  <User className="w-3 h-3 text-dark-500" />
                                  <span className="text-xs text-dark-400">{author}</span>
                                </div>
                              )}
                            </div>
                          </div>
                        </div>
                      </div>
                      
                      <p className="text-sm text-dark-300 mb-4 line-clamp-2 min-h-[40px]">
                        {getPluginDescription(plugin) || 'A network analysis plugin for NetTool'}
                      </p>

                      {/* Category/Tags */}
                      {(category || tags.length > 0) && (
                        <div className="flex flex-wrap gap-1 mb-4">
                          {category && (
                            <span className="px-2 py-0.5 text-xs rounded-full bg-dark-700/50 text-dark-300">
                              {category}
                            </span>
                          )}
                          {tags.slice(0, 2).map((tag, i) => (
                            <span key={i} className="px-2 py-0.5 text-xs rounded-full bg-dark-700/50 text-dark-400">
                              {tag}
                            </span>
                          ))}
                        </div>
                      )}

                      <div className="flex items-center gap-2" onClick={e => e.stopPropagation()}>
                        <button
                          onClick={() => handleInstall(repository)}
                          disabled={actionLoading[repository] || !repository}
                          className="btn-primary text-sm flex-1 disabled:opacity-50"
                        >
                          {actionLoading[repository] ? (
                            <RefreshCw className="w-4 h-4 animate-spin mx-auto" />
                          ) : (
                            <>
                              <Download className="w-4 h-4 inline mr-1" />
                              Install
                            </>
                          )}
                        </button>
                        {repository && (
                          <a
                            href={repository}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="p-2 rounded-lg bg-dark-800/50 text-dark-400 hover:text-white hover:bg-dark-700/50 transition-colors"
                            title="View on GitHub"
                          >
                            <Github className="w-4 h-4" />
                          </a>
                        )}
                      </div>
                    </motion.div>
                  )
                })
              ) : (
                <div className="col-span-full text-center py-12 text-dark-400">
                  <Package className="w-12 h-12 mx-auto mb-4 opacity-50" />
                  <p>No plugins available</p>
                  <p className="text-sm mt-1">Click refresh to check for new plugins</p>
                </div>
              )}
            </div>
          )}
        </>
      )}
    </div>
  )
}
