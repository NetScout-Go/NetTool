import { NavLink } from 'react-router-dom'
import { useState, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { 
  LayoutDashboard, 
  Puzzle, 
  Network, 
  Search, 
  Globe, 
  Shield, 
  Gauge,
  Wifi,
  Activity,
  Radar,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  Sun,
  Moon,
  Settings,
  RefreshCw
} from 'lucide-react'
import { usePlugins } from '../../context/PluginContext'
import { PLUGIN_CATEGORIES, getPluginCategory } from '../../types/plugins'

// Icon mapping
const iconMap = {
  Activity,
  Search,
  Network,
  Gauge,
  Globe,
  Shield,
  Wifi,
  Puzzle,
}

export default function Sidebar({ isOpen, onToggle, darkMode, onDarkModeToggle }) {
  const { plugins, loading, loadPlugins } = usePlugins()
  const [expandedGroups, setExpandedGroups] = useState({})

  // Normalize plugin data to handle both PascalCase and lowercase JSON keys
  const normalizedPlugins = useMemo(() => {
    return plugins.map(plugin => ({
      ID: plugin.ID || plugin.id,
      Name: plugin.Name || plugin.name,
      Description: plugin.Description || plugin.description,
      Version: plugin.Version || plugin.version,
      Author: plugin.Author || plugin.author,
      Icon: plugin.Icon || plugin.icon,
    }))
  }, [plugins])

  // Group plugins by category
  const pluginsByCategory = useMemo(() => {
    const grouped = {}
    
    // Initialize with empty arrays for all categories
    Object.keys(PLUGIN_CATEGORIES).forEach(categoryId => {
      grouped[categoryId] = []
    })
    
    // Group plugins
    normalizedPlugins.forEach(plugin => {
      const categoryId = getPluginCategory(plugin.ID)
      if (categoryId && grouped[categoryId]) {
        grouped[categoryId].push(plugin)
      }
    })
    
    return grouped
  }, [normalizedPlugins])

  // Get categories with plugins
  const categoriesWithPlugins = useMemo(() => {
    return Object.entries(PLUGIN_CATEGORIES)
      .filter(([categoryId]) => pluginsByCategory[categoryId]?.length > 0)
      .map(([categoryId, category]) => ({
        ...category,
        plugins: pluginsByCategory[categoryId],
      }))
  }, [pluginsByCategory])

  const toggleGroup = (groupId) => {
    setExpandedGroups(prev => ({
      ...prev,
      [groupId]: !prev[groupId]
    }))
  }

  return (
    <motion.aside
      initial={false}
      animate={{ width: isOpen ? 256 : 64 }}
      className="fixed left-0 top-0 h-full z-50"
    >
      <div className="h-full glass-card rounded-none border-r border-dark-800/50 flex flex-col">
        {/* Logo */}
        <div className="flex items-center justify-between p-4 border-b border-dark-800/50">
          <AnimatePresence mode="wait">
            {isOpen && (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                className="flex items-center gap-3"
              >
                <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-primary-500 to-primary-600 flex items-center justify-center">
                  <Wifi className="w-5 h-5 text-white" />
                </div>
                <span className="text-lg font-bold text-gradient">NetTool</span>
              </motion.div>
            )}
          </AnimatePresence>
          
          <button
            onClick={onToggle}
            className="p-2 rounded-lg hover:bg-dark-800/50 transition-colors"
          >
            {isOpen ? <ChevronLeft className="w-5 h-5" /> : <ChevronRight className="w-5 h-5" />}
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 overflow-y-auto py-4 px-2 space-y-2">
          {/* Dashboard */}
          <NavLink
            to="/"
            className={({ isActive }) => `
              flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-200
              ${isActive 
                ? 'bg-primary-500/20 text-primary-400 border border-primary-500/30' 
                : 'hover:bg-dark-800/50 text-dark-300 hover:text-white'}
            `}
          >
            <LayoutDashboard className="w-5 h-5 flex-shrink-0" />
            {isOpen && <span className="font-medium">Dashboard</span>}
          </NavLink>

          {/* Plugin Manager */}
          <NavLink
            to="/plugin-manager"
            className={({ isActive }) => `
              flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-200
              ${isActive 
                ? 'bg-primary-500/20 text-primary-400 border border-primary-500/30' 
                : 'hover:bg-dark-800/50 text-dark-300 hover:text-white'}
            `}
          >
            <Settings className="w-5 h-5 flex-shrink-0" />
            {isOpen && <span className="font-medium">Plugin Manager</span>}
          </NavLink>

          {/* Insights */}
          <NavLink
            to="/insights"
            className={({ isActive }) => `
              flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-200
              ${isActive 
                ? 'bg-primary-500/20 text-primary-400 border border-primary-500/30' 
                : 'hover:bg-dark-800/50 text-dark-300 hover:text-white'}
            `}
          >
            <Radar className="w-5 h-5 flex-shrink-0" />
            {isOpen && <span className="font-medium">Insights</span>}
          </NavLink>

          {/* Divider */}
          <div className="my-4 border-t border-dark-800/50" />

          {/* Loading indicator */}
          {loading && isOpen && (
            <div className="flex items-center justify-center py-4">
              <RefreshCw className="w-5 h-5 text-primary-400 animate-spin" />
              <span className="ml-2 text-sm text-dark-400">Loading plugins...</span>
            </div>
          )}

          {/* Plugin Groups */}
          {isOpen && !loading && (
            <div className="space-y-1">
              {categoriesWithPlugins.length > 0 ? (
                categoriesWithPlugins.map((category) => {
                  const Icon = iconMap[category.icon] || Puzzle
                  const isExpanded = expandedGroups[category.id]

                  return (
                    <div key={category.id}>
                      <button
                        onClick={() => toggleGroup(category.id)}
                        className="w-full flex items-center justify-between px-3 py-2.5 rounded-xl hover:bg-dark-800/50 text-dark-300 hover:text-white transition-all duration-200"
                      >
                        <div className="flex items-center gap-3">
                          <Icon className="w-5 h-5" />
                          <span className="font-medium text-sm">{category.label}</span>
                          <span className="text-xs text-dark-500">({category.plugins.length})</span>
                        </div>
                        <motion.div
                          animate={{ rotate: isExpanded ? 180 : 0 }}
                          transition={{ duration: 0.2 }}
                        >
                          <ChevronDown className="w-4 h-4" />
                        </motion.div>
                      </button>

                      <AnimatePresence>
                        {isExpanded && (
                          <motion.div
                            initial={{ opacity: 0, height: 0 }}
                            animate={{ opacity: 1, height: 'auto' }}
                            exit={{ opacity: 0, height: 0 }}
                            transition={{ duration: 0.2 }}
                            className="overflow-hidden"
                          >
                            <div className="pl-4 space-y-1 mt-1">
                              {category.plugins.map((plugin) => (
                                <NavLink
                                  key={plugin.ID}
                                  to={`/plugin/${plugin.ID}`}
                                  className={({ isActive }) => `
                                    flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-all duration-200
                                    ${isActive 
                                      ? 'bg-primary-500/10 text-primary-400' 
                                      : 'hover:bg-dark-800/30 text-dark-400 hover:text-white'}
                                  `}
                                >
                                  <Puzzle className="w-4 h-4" />
                                  <span className="truncate">{plugin.Name}</span>
                                </NavLink>
                              ))}
                            </div>
                          </motion.div>
                        )}
                      </AnimatePresence>
                    </div>
                  )
                })
              ) : (
                <div className="px-3 py-4 text-center">
                  <Puzzle className="w-8 h-8 text-dark-600 mx-auto mb-2" />
                  <p className="text-sm text-dark-400">No plugins installed</p>
                  <NavLink
                    to="/plugin-manager"
                    className="text-xs text-primary-400 hover:text-primary-300 mt-1 inline-block"
                  >
                    Browse plugins →
                  </NavLink>
                </div>
              )}
            </div>
          )}

          {/* Collapsed view - just show icons */}
          {!isOpen && !loading && categoriesWithPlugins.length > 0 && (
            <div className="space-y-2">
              {categoriesWithPlugins.map((category) => {
                const Icon = iconMap[category.icon] || Puzzle
                return (
                  <div
                    key={category.id}
                    className="flex items-center justify-center p-2.5 rounded-xl text-dark-400 hover:bg-dark-800/50 hover:text-white transition-colors cursor-pointer"
                    title={`${category.label} (${category.plugins.length})`}
                  >
                    <Icon className="w-5 h-5" />
                  </div>
                )
              })}
            </div>
          )}
        </nav>

        {/* Footer */}
        <div className="p-4 border-t border-dark-800/50">
          <button
            onClick={onDarkModeToggle}
            className="w-full flex items-center justify-center gap-2 px-3 py-2.5 rounded-xl hover:bg-dark-800/50 text-dark-300 hover:text-white transition-all duration-200"
          >
            {darkMode ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
            {isOpen && <span className="text-sm">Toggle Theme</span>}
          </button>
          
          {isOpen && (
            <div className="mt-4 text-center text-xs text-dark-500">
              <p>NetTool v2.0</p>
              <p className="mt-1">Network Analysis Tool</p>
            </div>
          )}
        </div>
      </div>
    </motion.aside>
  )
}
