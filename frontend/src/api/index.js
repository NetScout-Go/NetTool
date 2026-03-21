import axios from 'axios'

// API Error class for better error handling
export class ApiError extends Error {
  constructor(message, code, status, details = null) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
    this.details = details
  }
}

// Create axios instance with default config
const api = axios.create({
  baseURL: '/api',
  timeout: 60000, // Increased for longer plugin operations
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor for logging and adding headers
api.interceptors.request.use(
  (config) => {
    // Add timestamp to requests for debugging
    config.metadata = { startTime: new Date() }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => {
    // Calculate request duration
    const duration = new Date() - response.config.metadata.startTime
    response.duration = duration
    return response
  },
  (error) => {
    // Handle different error scenarios
    if (error.response) {
      // Server responded with error
      const { status, data } = error.response
      const message = data?.error || data?.message || 'An error occurred'
      const code = data?.code || `HTTP_${status}`
      throw new ApiError(message, code, status, data?.details)
    } else if (error.request) {
      // Request made but no response
      throw new ApiError('No response from server', 'NETWORK_ERROR', 0)
    } else {
      // Request setup error
      throw new ApiError(error.message, 'REQUEST_ERROR', 0)
    }
  }
)

// Network API
export const networkApi = {
  getNetworkInfo: () => api.get('/network-info'),
}

// Diagnostics / history API
export const diagnosticsApi = {
  getSummary: () => api.get('/diagnostics/summary'),
  getHistory: (limit = 60) => api.get(`/network-history?limit=${limit}`),
  getHistoryExportUrl: (format = 'csv', limit = 120) => `/api/network-history/export?format=${format}&limit=${limit}`,
}

// Interface Detection API
export const interfacesApi = {
  // Get all interfaces categorized
  getAll: () => api.get('/interfaces'),
  
  // Get primary WiFi interface
  getPrimaryWiFi: () => api.get('/interfaces/wifi'),
  
  // Get primary Ethernet interface
  getPrimaryEthernet: () => api.get('/interfaces/ethernet'),
  
  // Get interfaces by type (ethernet, wifi, loopback, virtual, bridge, vpn)
  getByType: (type) => api.get(`/interfaces/type/${type}`),
}

// Plugins API with enhanced functionality
export const pluginsApi = {
  // Get all registered plugins
  getAll: () => api.get('/plugins'),
  
  // Get specific plugin by ID
  getById: (id) => api.get(`/plugins/${id}`),
  
  // Run a plugin with parameters
  run: (id, params = {}, config = {}) => {
    const { timeout = 60000 } = config
    return api.post(`/plugins/${id}/run`, params, { timeout })
  },
  
  // Generic plugin runner
  runPlugin: (id, params = {}) => api.post('/run-plugin', { id, params }),
  
  // Run plugin with iteration support
  runWithIteration: (id, params = {}, iterationConfig = {}) => {
    const payload = {
      ...params,
      _config: {
        iterate: iterationConfig.iterate || false,
        maxIterations: iterationConfig.maxIterations || 0,
        iterationDelay: iterationConfig.iterationDelay || 1000,
        continueOnError: iterationConfig.continueOnError || false,
      },
    }
    return api.post(`/plugins/${id}/run`, payload, { timeout: 300000 })
  },
}

// Plugin Manager API with enhanced functionality
export const pluginManagerApi = {
  // List all installed plugins
  listInstalled: () => api.get('/plugins/manage/list'),
  
  // Get detailed plugin information
  getDetails: (id) => api.get(`/plugins/manage/details/${id}`),
  
  // List available plugins from GitHub
  listAvailable: () => api.get('/plugins/manage/available'),
  
  // Refresh plugin catalog from GitHub
  refreshCatalog: () => api.post('/plugins/manage/refresh-catalog'),
  
  // Check if prebuilt binaries are available for a plugin
  checkAvailability: (repository) => api.post('/plugins/manage/check-availability', { repository }),
  
  // Check availability for multiple plugins
  checkAvailabilityBulk: (repositories) => api.post('/plugins/manage/check-availability-bulk', { repositories }),
  
  // Install plugin from repository with channel selection
  // channel: 'stable' (default), 'beta', or 'source'
  install: (repository, channel = 'stable') => api.post('/plugins/manage/install', { repository, channel }),
  
  // Bulk install multiple plugins with channel selection
  bulkInstall: (repositories, channel = 'stable') => api.post('/plugins/manage/bulk-install', { repositories, channel }),
  
  // Update a specific plugin
  update: (id) => api.post(`/plugins/manage/update/${id}`),
  
  // Uninstall a plugin
  uninstall: (id) => api.post(`/plugins/manage/uninstall/${id}`),
  
  // Update all plugins with available updates
  updateAll: () => api.post('/plugins/manage/update-all'),
  
  // Sync version information with GitHub
  sync: () => api.post('/plugins/manage/sync'),
  
  // Check if file exists (for README, etc.)
  fileExists: (path) => api.get('/plugins/manage/file-exists', { params: { path } }),
  
  // View file contents
  viewFile: (path) => api.get('/plugins/manage/view-file', { params: { path } }),
}

// Utility functions
export const apiUtils = {
  // Check if error is an API error
  isApiError: (error) => error instanceof ApiError,
  
  // Get user-friendly error message
  getErrorMessage: (error) => {
    if (error instanceof ApiError) {
      return error.message
    }
    return error?.message || 'An unexpected error occurred'
  },
  
  // Check if error is network related
  isNetworkError: (error) => {
    return error instanceof ApiError && error.code === 'NETWORK_ERROR'
  },
}

export default api
