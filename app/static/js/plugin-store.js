/**
 * Plugin Store UI Extensions for NetTool Plugin Manager
 */

// Extend the PluginManagerUI with store functionality
Object.assign(PluginManagerUI, {
    // Initialize store functionality
    initStore: function() {
        // Load available plugins when the store tab is shown
        document.getElementById('store-tab').addEventListener('shown.bs.tab', () => {
            this.loadAvailablePlugins();
        });
        
        // Set up event listeners for store functionality
        this.setupStoreEventListeners();
    },
    
    // Set up event listeners for store functionality
    setupStoreEventListeners: function() {
        // Refresh catalog button
        const refreshCatalogBtn = document.getElementById('refreshCatalogBtn');
        if (refreshCatalogBtn) {
            refreshCatalogBtn.addEventListener('click', () => {
                this.refreshPluginCatalog();
            });
        }
        
        // Search input
        const searchInput = document.getElementById('storeSearchInput');
        if (searchInput) {
            searchInput.addEventListener('input', () => {
                this.filterPluginStore();
            });
        }
        
        // Category filter
        const categoryFilter = document.getElementById('categoryFilter');
        if (categoryFilter) {
            categoryFilter.addEventListener('change', () => {
                this.filterPluginStore();
            });
        }
        
        // Show installed toggle
        const showInstalledToggle = document.getElementById('showInstalledToggle');
        if (showInstalledToggle) {
            showInstalledToggle.addEventListener('change', () => {
                this.filterPluginStore();
            });
        }
        
        // Install button in details modal
        const installBtn = document.getElementById('installPluginFromStoreBtn');
        if (installBtn) {
            installBtn.addEventListener('click', () => {
                const pluginId = installBtn.getAttribute('data-plugin-id');
                if (pluginId) {
                    this.installPluginFromStore(pluginId);
                }
            });
        }
    },
    
    // Load available plugins from the server
    loadAvailablePlugins: function() {
        const storeGrid = document.getElementById('pluginStoreGrid');
        storeGrid.innerHTML = `
            <div class="col-12 text-center py-5">
                <div class="spinner-border text-primary" role="status">
                    <span class="visually-hidden">Loading...</span>
                </div>
                <p class="mt-3">Loading plugin catalog...</p>
            </div>
        `;
        
        // Fetch available plugins
        fetch('/api/plugins/manage/available')
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to load plugin catalog');
                }
                return response.json();
            })
            .then(plugins => {
                this.renderPluginStore(plugins);
            })
            .catch(error => {
                console.error('Error loading plugin catalog:', error);
                storeGrid.innerHTML = `
                    <div class="col-12 text-center py-5">
                        <div class="alert alert-danger">
                            <i class="bi bi-exclamation-triangle me-2"></i>
                            Failed to load plugin catalog: ${error.message}
                        </div>
                        <button class="btn btn-primary mt-3" id="retryLoadCatalogBtn">
                            <i class="bi bi-arrow-clockwise me-2"></i>Retry
                        </button>
                    </div>
                `;
                
                // Add retry button handler
                document.getElementById('retryLoadCatalogBtn').addEventListener('click', () => {
                    this.loadAvailablePlugins();
                });
                
                this.showToast('Error', 'Failed to load plugin catalog: ' + error.message, 'error');
            });
    },
    
    // Refresh the plugin catalog from GitHub
    refreshPluginCatalog: function() {
        const refreshBtn = document.getElementById('refreshCatalogBtn');
        refreshBtn.disabled = true;
        refreshBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Refreshing...';
        
        // Call the API to refresh the catalog
        fetch('/api/plugins/manage/refresh-catalog', {
            method: 'POST'
        })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to refresh plugin catalog');
                }
                return response.json();
            })
            .then(data => {
                this.showToast('Success', 'Plugin catalog refreshed successfully', 'success');
                this.loadAvailablePlugins();
            })
            .catch(error => {
                console.error('Error refreshing plugin catalog:', error);
                this.showToast('Error', 'Failed to refresh plugin catalog: ' + error.message, 'error');
            })
            .finally(() => {
                refreshBtn.disabled = false;
                refreshBtn.innerHTML = '<i class="bi bi-cloud-download me-1"></i> Refresh Catalog';
            });
    },
    
    // Render the plugin store with available plugins
    renderPluginStore: function(plugins) {
        const storeGrid = document.getElementById('pluginStoreGrid');
        
        if (!plugins || plugins.length === 0) {
            storeGrid.innerHTML = `
                <div class="col-12 text-center py-5">
                    <div class="alert alert-info">
                        <i class="bi bi-info-circle me-2"></i>
                        No plugins found in the catalog
                    </div>
                    <button class="btn btn-primary mt-3" id="refreshEmptyCatalogBtn">
                        <i class="bi bi-cloud-download me-2"></i>Refresh Catalog
                    </button>
                </div>
            `;
            
            // Add refresh button handler
            document.getElementById('refreshEmptyCatalogBtn').addEventListener('click', () => {
                this.refreshPluginCatalog();
            });
            
            return;
        }
        
        // Store plugins for filtering
        this.availablePlugins = plugins;
        
        // Render plugins
        this.filterPluginStore();
    },
    
    // Filter plugins in the store based on search, category, and installed status
    filterPluginStore: function() {
        if (!this.availablePlugins) return;
        
        const searchInput = document.getElementById('storeSearchInput');
        const categoryFilter = document.getElementById('categoryFilter');
        const showInstalledToggle = document.getElementById('showInstalledToggle');
        
        const searchTerm = searchInput.value.toLowerCase();
        const category = categoryFilter.value;
        const showInstalled = showInstalledToggle.checked;
        
        // Filter plugins
        const filteredPlugins = this.availablePlugins.filter(plugin => {
            // Filter by search term
            const matchesSearch = 
                plugin.name.toLowerCase().includes(searchTerm) || 
                plugin.description.toLowerCase().includes(searchTerm) ||
                plugin.id.toLowerCase().includes(searchTerm);
            
            // Filter by category
            const matchesCategory = !category || plugin.category === category;
            
            // Filter by installed status
            const matchesInstalled = showInstalled || !plugin.installed;
            
            return matchesSearch && matchesCategory && matchesInstalled;
        });
        
        // Render filtered plugins
        const storeGrid = document.getElementById('pluginStoreGrid');
        
        if (filteredPlugins.length === 0) {
            storeGrid.innerHTML = `
                <div class="col-12 text-center py-5">
                    <div class="alert alert-info">
                        <i class="bi bi-info-circle me-2"></i>
                        No plugins match your search criteria
                    </div>
                </div>
            `;
            return;
        }
        
        storeGrid.innerHTML = filteredPlugins.map(plugin => this.createPluginCard(plugin)).join('');
        
        // Add event listeners to the plugin cards
        storeGrid.querySelectorAll('.plugin-card').forEach(card => {
            const pluginId = card.getAttribute('data-plugin-id');
            
            // Details button
            card.querySelector('.plugin-details-btn').addEventListener('click', () => {
                this.showPluginStoreDetails(pluginId);
            });
            
            // Install button
            const installBtn = card.querySelector('.plugin-install-btn');
            if (installBtn) {
                installBtn.addEventListener('click', () => {
                    this.installPluginFromStore(pluginId);
                });
            }
        });
    },
    
    // Create a plugin card for the store
    createPluginCard: function(plugin) {
        const installButton = plugin.installed 
            ? `<button class="btn btn-sm btn-outline-success" disabled>
                <i class="bi bi-check-circle me-1"></i>Installed
               </button>`
            : `<button class="btn btn-sm btn-primary plugin-install-btn">
                <i class="bi bi-download me-1"></i>Install
               </button>`;
        
        return `
            <div class="col">
                <div class="card h-100 plugin-card" data-plugin-id="${plugin.id}">
                    <div class="card-header bg-light d-flex align-items-center">
                        <i class="bi bi-${plugin.icon || 'plugin'} me-2 fs-5"></i>
                        <div>
                            <h5 class="card-title mb-0">${plugin.name}</h5>
                            <div class="small text-muted">${plugin.id}</div>
                        </div>
                    </div>
                    <div class="card-body">
                        <p class="card-text">${plugin.description}</p>
                        <div class="plugin-meta">
                            <span class="badge bg-primary me-1">${plugin.category || 'other'}</span>
                            <span class="badge bg-secondary me-1">v${plugin.version}</span>
                            <span class="badge bg-light text-dark">${plugin.author}</span>
                        </div>
                    </div>
                    <div class="card-footer d-flex justify-content-between align-items-center">
                        <button class="btn btn-sm btn-outline-secondary plugin-details-btn">
                            <i class="bi bi-info-circle me-1"></i>Details
                        </button>
                        ${installButton}
                    </div>
                </div>
            </div>
        `;
    },
    
    // Show plugin details from the store
    showPluginStoreDetails: function(pluginId) {
        const plugin = this.availablePlugins.find(p => p.id === pluginId);
        if (!plugin) return;
        
        const modalTitle = document.getElementById('pluginDetailsModalLabel');
        const modalContent = document.querySelector('.plugin-details-content');
        const installBtn = document.getElementById('installPluginFromStoreBtn');
        const uninstallBtn = document.getElementById('uninstallPluginBtn');
        const updateBtn = document.getElementById('updatePluginBtn');
        
        modalTitle.textContent = plugin.name;
        
        // Format and display plugin details
        let html = '<div class="plugin-details">';
        
        // Plugin Info Card
        html += `
            <div class="card mb-4">
                <div class="card-header bg-light">
                    <h6 class="mb-0"><i class="bi bi-info-circle me-2"></i>Plugin Information</h6>
                </div>
                <div class="card-body">
                    <div class="row mb-3">
                        <div class="col-md-4 fw-bold">ID:</div>
                        <div class="col-md-8">${plugin.id}</div>
                    </div>
                    <div class="row mb-3">
                        <div class="col-md-4 fw-bold">Description:</div>
                        <div class="col-md-8">${plugin.description}</div>
                    </div>
                    <div class="row mb-3">
                        <div class="col-md-4 fw-bold">Version:</div>
                        <div class="col-md-8">${plugin.version}</div>
                    </div>
                    <div class="row mb-3">
                        <div class="col-md-4 fw-bold">Author:</div>
                        <div class="col-md-8">${plugin.author}</div>
                    </div>
                    <div class="row mb-3">
                        <div class="col-md-4 fw-bold">License:</div>
                        <div class="col-md-8">${plugin.license}</div>
                    </div>
                    <div class="row mb-3">
                        <div class="col-md-4 fw-bold">Category:</div>
                        <div class="col-md-8">
                            <span class="badge bg-primary">${plugin.category || 'other'}</span>
                        </div>
                    </div>
                    <div class="row">
                        <div class="col-md-4 fw-bold">Repository:</div>
                        <div class="col-md-8">
                            <a href="${plugin.repository}" target="_blank">${plugin.repository}</a>
                        </div>
                    </div>
                </div>
            </div>
        `;
        
        html += '</div>';
        
        modalContent.innerHTML = html;
        
        // Set up modal buttons
        if (plugin.installed) {
            installBtn.classList.add('d-none');
            uninstallBtn.classList.remove('d-none');
            updateBtn.classList.remove('d-none');
        } else {
            installBtn.classList.remove('d-none');
            uninstallBtn.classList.add('d-none');
            updateBtn.classList.add('d-none');
            
            // Set plugin ID for install button
            installBtn.setAttribute('data-plugin-id', plugin.id);
        }
        
        // Set repository URL for README button
        const readmeBtn = document.getElementById('viewPluginReadme');
        readmeBtn.href = `${plugin.repository}/blob/main/README.md`;
        
        // Show the modal
        const modal = new bootstrap.Modal(document.getElementById('pluginDetailsModal'));
        modal.show();
    },
    
    // Install a plugin from the store
    installPluginFromStore: function(pluginId) {
        const plugin = this.availablePlugins.find(p => p.id === pluginId);
        if (!plugin) return;
        
        this.confirmAction(
            `Are you sure you want to install the plugin "${plugin.name}"?`,
            () => {
                // Hide the modal
                const modal = bootstrap.Modal.getInstance(document.getElementById('pluginDetailsModal'));
                modal.hide();
                
                // Show toast
                this.showToast('Installing Plugin', `Installing ${plugin.name}...`, 'info');
                
                // Install the plugin
                fetch('/api/plugins/manage/install', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        repository: plugin.repository
                    })
                })
                    .then(response => {
                        if (!response.ok) {
                            throw new Error('Failed to install plugin');
                        }
                        return response.json();
                    })
                    .then(data => {
                        this.showToast('Success', `Plugin ${plugin.name} installed successfully`, 'success');
                        
                        // Mark plugin as installed
                        plugin.installed = true;
                        
                        // Refresh the plugin store
                        this.filterPluginStore();
                        
                        // Refresh the installed plugins
                        this.loadInstalledPlugins();
                    })
                    .catch(error => {
                        console.error('Error installing plugin:', error);
                        this.showToast('Error', 'Failed to install plugin: ' + error.message, 'error');
                    });
            }
        );
    }
});

// Add store initialization to the main init function
const originalInit = PluginManagerUI.init;
PluginManagerUI.init = function() {
    originalInit.call(this);
    this.initStore();
};
