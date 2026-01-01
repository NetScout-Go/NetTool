/**
 * Plugin type definitions for NetTool frontend
 * These types mirror the Go backend types for consistency
 */

/**
 * Parameter types supported by plugins
 * @typedef {'string' | 'number' | 'boolean' | 'select' | 'range'} ParameterType
 */

/**
 * Option for select parameters
 * @typedef {Object} SelectOption
 * @property {string|number} value - The option value
 * @property {string} label - Display label
 */

/**
 * Plugin parameter definition
 * @typedef {Object} PluginParameter
 * @property {string} id - Unique parameter identifier
 * @property {string} name - Display name
 * @property {string} description - Help text
 * @property {ParameterType} type - Parameter type
 * @property {boolean} required - Whether parameter is required
 * @property {*} [default] - Default value
 * @property {SelectOption[]} [options] - Options for select type
 * @property {number} [min] - Minimum value for number/range
 * @property {number} [max] - Maximum value for number/range
 * @property {number} [step] - Step value for number/range
 * @property {boolean} [canIterate] - Whether parameter supports iteration
 */

/**
 * Plugin definition
 * @typedef {Object} Plugin
 * @property {string} ID - Plugin identifier
 * @property {string} Name - Display name
 * @property {string} Description - Plugin description
 * @property {string} Version - Version string
 * @property {string} Author - Author name
 * @property {string} License - License type
 * @property {string} Icon - Icon name
 * @property {PluginParameter[]} Parameters - Plugin parameters
 * @property {string[]} [Requires] - System dependencies
 * @property {string} [Repository] - GitHub repository URL
 */

/**
 * Plugin execution result
 * @typedef {Object} PluginResult
 * @property {*} data - Result data
 * @property {boolean} success - Whether execution succeeded
 * @property {string} [error] - Error message if failed
 * @property {Date} timestamp - Execution timestamp
 */

/**
 * Plugin execution state
 * @typedef {Object} PluginRunState
 * @property {boolean} running - Whether plugin is running
 * @property {number} progress - Progress percentage (0-100)
 * @property {string} message - Status message
 */

/**
 * Installed plugin metadata
 * @typedef {Object} InstalledPlugin
 * @property {string} ID - Plugin identifier
 * @property {string} Name - Display name
 * @property {string} Description - Plugin description
 * @property {string} Version - Installed version
 * @property {string} Author - Author name
 * @property {string} License - License type
 * @property {string} Icon - Icon name
 * @property {string} Status - Plugin status
 * @property {boolean} UpdateAvailable - Whether update is available
 * @property {string} [LatestVersion] - Latest available version
 * @property {string} Path - Installation path
 * @property {GitInfo} [GitInfo] - Git repository information
 */

/**
 * Git version information
 * @typedef {Object} GitInfo
 * @property {string} CommitID - Current commit ID
 * @property {string} Branch - Current branch
 * @property {string} [LatestCommitID] - Latest remote commit
 * @property {string} [Repository] - Repository name
 * @property {string} [Organization] - GitHub organization
 */

/**
 * Available plugin from store
 * @typedef {Object} AvailablePlugin
 * @property {string} ID - Plugin identifier
 * @property {string} Name - Display name
 * @property {string} Description - Plugin description
 * @property {string} Version - Available version
 * @property {string} Author - Author name
 * @property {string} License - License type
 * @property {string} Category - Plugin category
 * @property {string} Repository - GitHub repository URL
 * @property {string} Icon - Icon name
 * @property {boolean} Installed - Whether already installed
 * @property {string[]} [Screenshots] - Screenshot URLs
 * @property {string[]} [Requirements] - System requirements
 * @property {string[]} [Tags] - Search tags
 */

/**
 * Plugin categories with their metadata
 */
export const PLUGIN_CATEGORIES = {
  'network-analysis': {
    id: 'network-analysis',
    label: 'Network Analysis',
    icon: 'Activity',
    description: 'Traffic analysis, bandwidth testing, and network metrics',
    pluginIds: ['network_quality', 'bandwidth_test', 'packet_capture', 'network_info', 'iperf3', 'tc_controller', 'network_latency_heatmap', 'subnet_calculator'],
  },
  'network-discovery': {
    id: 'network-discovery',
    label: 'Network Discovery',
    icon: 'Search',
    description: 'Device discovery and port scanning',
    pluginIds: ['device_discovery', 'port_scanner', 'wifi_scanner'],
  },
  'connectivity': {
    id: 'connectivity',
    label: 'Connectivity',
    icon: 'Network',
    description: 'Connection testing and path tracing',
    pluginIds: ['ping', 'traceroute', 'mtu_tester'],
  },
  'performance': {
    id: 'performance',
    label: 'Performance',
    icon: 'Gauge',
    description: 'Speed tests and performance monitoring',
    pluginIds: ['iperf3', 'iperf3_server', 'tc_controller'],
  },
  'dns-tools': {
    id: 'dns-tools',
    label: 'DNS Tools',
    icon: 'Globe',
    description: 'DNS lookup and propagation checks',
    pluginIds: ['dns_lookup', 'dns_propagation', 'reverse_dns_lookup'],
  },
  'security': {
    id: 'security',
    label: 'Security',
    icon: 'Shield',
    description: 'SSL certificates and security scanning',
    pluginIds: ['ssl_checker'],
  },
}

/**
 * Get category for a plugin ID
 * @param {string} pluginId - Plugin identifier
 * @returns {string|null} Category ID or null if not found
 */
export function getPluginCategory(pluginId) {
  for (const [categoryId, category] of Object.entries(PLUGIN_CATEGORIES)) {
    if (category.pluginIds.includes(pluginId)) {
      return categoryId
    }
  }
  return null
}

/**
 * Get category metadata
 * @param {string} categoryId - Category identifier
 * @returns {Object|null} Category metadata or null
 */
export function getCategoryInfo(categoryId) {
  return PLUGIN_CATEGORIES[categoryId] || null
}

/**
 * Icon mapping for plugins
 */
export const PLUGIN_ICONS = {
  network: 'Network',
  ping: 'Activity',
  dns: 'Globe',
  port: 'Server',
  route: 'GitBranch',
  speed: 'Gauge',
  info: 'Info',
  scan: 'Search',
  security: 'Shield',
  wifi: 'Wifi',
  default: 'Puzzle',
}

/**
 * Get icon component name for a plugin
 * @param {string} iconName - Icon name from plugin
 * @returns {string} Lucide icon component name
 */
export function getPluginIcon(iconName) {
  return PLUGIN_ICONS[iconName] || PLUGIN_ICONS.default
}

/**
 * Parameter type configurations
 */
export const PARAMETER_TYPES = {
  string: {
    type: 'text',
    component: 'input',
    validate: (value, param) => {
      if (param.required && !value) {
        return 'This field is required'
      }
      return null
    },
  },
  number: {
    type: 'number',
    component: 'input',
    validate: (value, param) => {
      if (param.required && (value === '' || value === null || value === undefined)) {
        return 'This field is required'
      }
      const num = Number(value)
      if (isNaN(num)) {
        return 'Must be a valid number'
      }
      if (param.min !== undefined && num < param.min) {
        return `Must be at least ${param.min}`
      }
      if (param.max !== undefined && num > param.max) {
        return `Must be at most ${param.max}`
      }
      return null
    },
  },
  boolean: {
    type: 'checkbox',
    component: 'checkbox',
    validate: () => null,
  },
  select: {
    type: 'select',
    component: 'select',
    validate: (value, param) => {
      if (param.required && !value) {
        return 'Please select an option'
      }
      return null
    },
  },
  range: {
    type: 'range',
    component: 'slider',
    validate: (value, param) => {
      const num = Number(value)
      if (param.min !== undefined && num < param.min) {
        return `Must be at least ${param.min}`
      }
      if (param.max !== undefined && num > param.max) {
        return `Must be at most ${param.max}`
      }
      return null
    },
  },
}

/**
 * Validate plugin parameters
 * @param {PluginParameter[]} parameters - Plugin parameter definitions
 * @param {Object} values - Parameter values
 * @returns {Object} Validation errors by parameter ID
 */
export function validateParameters(parameters, values) {
  const errors = {}
  
  for (const param of parameters) {
    const value = values[param.id]
    const typeConfig = PARAMETER_TYPES[param.type]
    
    if (typeConfig?.validate) {
      const error = typeConfig.validate(value, param)
      if (error) {
        errors[param.id] = error
      }
    }
  }
  
  return errors
}

/**
 * Initialize parameter values with defaults
 * @param {PluginParameter[]} parameters - Plugin parameter definitions
 * @returns {Object} Initial values object
 */
export function initializeParameters(parameters) {
  const values = {}
  
  for (const param of parameters || []) {
    if (param.default !== undefined) {
      values[param.id] = param.default
    } else {
      // Set sensible defaults based on type
      switch (param.type) {
        case 'boolean':
          values[param.id] = false
          break
        case 'number':
        case 'range':
          values[param.id] = param.min ?? 0
          break
        case 'select':
          values[param.id] = param.options?.[0]?.value ?? ''
          break
        default:
          values[param.id] = ''
      }
    }
  }
  
  return values
}

export default {
  PLUGIN_CATEGORIES,
  PLUGIN_ICONS,
  PARAMETER_TYPES,
  getPluginCategory,
  getCategoryInfo,
  getPluginIcon,
  validateParameters,
  initializeParameters,
}
