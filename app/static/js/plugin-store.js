/**
 * Plugin Store UI Extensions for NetTool Plugin Manager
 */

// Extend the PluginManagerUI with store functionality
Object.assign(PluginManagerUI, {
    // Store selected plugins for bulk operations
    selectedPlugins: new Set(),
    
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
        
        // Bulk selection controls
        const selectAllBtn = document.getElementById('selectAllBtn');
        if (selectAllBtn) {
            selectAllBtn.addEventListener('click', () => {
                this.selectAllPlugins();
            });
        }
        
        const deselectAllBtn = document.getElementById('deselectAllBtn');
        if (deselectAllBtn) {
            deselectAllBtn.addEventListener('click', () => {
                this.deselectAllPlugins();
            });
        }
        
        const bulkInstallBtn = document.getElementById('bulkInstallBtn');
        if (bulkInstallBtn) {
            bulkInstallBtn.addEventListener('click', () => {
                this.bulkInstallPlugins();
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
                plugin.id.toLowerCase().includes(searchTerm) ||
                plugin.author.toLowerCase().includes(searchTerm) ||
                (plugin.tags && plugin.tags.some(tag => tag.toLowerCase().includes(searchTerm))) ||
                (plugin.requirements && plugin.requirements.some(req => req.toLowerCase().includes(searchTerm)));
            
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
            
            // Selection checkbox
            const checkbox = card.querySelector('.plugin-select-checkbox');
            if (checkbox) {
                checkbox.addEventListener('change', () => {
                    this.handlePluginSelection(checkbox);
                });
            }
        });
        
        // Update bulk action controls visibility
        this.updateBulkControls();
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

        // Add selection checkbox for non-installed plugins
        const selectionCheckbox = !plugin.installed 
            ? `<div class="form-check position-absolute top-0 start-0 m-2">
                    <input class="form-check-input plugin-select-checkbox" type="checkbox" value="${plugin.id}" data-repository="${plugin.repository}">
               </div>`
            : '';

        // Create tags display
        const tagsDisplay = plugin.tags && plugin.tags.length > 0 
            ? plugin.tags.slice(0, 3).map(tag => `<span class="badge bg-light text-dark me-1">${tag}</span>`).join('')
            : '';

        // Truncate description if too long
        const description = plugin.description.length > 120 
            ? plugin.description.substring(0, 120) + '...'
            : plugin.description;
        
        return `
            <div class="col">
                <div class="card h-100 plugin-card position-relative" data-plugin-id="${plugin.id}">
                    ${selectionCheckbox}
                    <div class="card-header bg-light d-flex align-items-center">
                        <i class="bi bi-${plugin.icon || 'plugin'} me-2 fs-5"></i>
                        <div class="flex-grow-1">
                            <h5 class="card-title mb-0">${plugin.name}</h5>
                            <div class="small text-muted">${plugin.id}</div>
                        </div>
                    </div>
                    <div class="card-body">
                        <p class="card-text">${description}</p>
                        <div class="plugin-meta mb-2">
                            <span class="badge bg-primary me-1">${plugin.category || 'other'}</span>
                            <span class="badge bg-secondary me-1">v${plugin.version}</span>
                            <span class="badge bg-info text-white">${plugin.author}</span>
                        </div>
                        ${tagsDisplay ? `<div class="plugin-tags">${tagsDisplay}</div>` : ''}
                        ${plugin.requirements && plugin.requirements.length > 0 ? 
                            `<div class="small text-muted mt-2">
                                <i class="bi bi-info-circle me-1"></i>Requirements: ${plugin.requirements.join(', ')}
                            </div>` : ''}
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
                            <a href="${plugin.repository}" target="_blank" rel="noopener noreferrer">
                                <i class="bi bi-github me-1"></i>${plugin.repository}
                            </a>
                        </div>
                    </div>
                </div>
            </div>
        `;

        // Tags section
        if (plugin.tags && plugin.tags.length > 0) {
            html += `
                <div class="card mb-4">
                    <div class="card-header bg-light">
                        <h6 class="mb-0"><i class="bi bi-tags me-2"></i>Tags</h6>
                    </div>
                    <div class="card-body">
                        ${plugin.tags.map(tag => `<span class="badge bg-secondary me-1 mb-1">${tag}</span>`).join('')}
                    </div>
                </div>
            `;
        }

        // Requirements section
        if (plugin.requirements && plugin.requirements.length > 0) {
            html += `
                <div class="card mb-4">
                    <div class="card-header bg-light">
                        <h6 class="mb-0"><i class="bi bi-exclamation-triangle me-2"></i>Requirements</h6>
                    </div>
                    <div class="card-body">
                        <ul class="list-unstyled mb-0">
                            ${plugin.requirements.map(req => `<li><i class="bi bi-check-circle-fill text-success me-2"></i>${req}</li>`).join('')}
                        </ul>
                    </div>
                </div>
            `;
        }

        // Screenshots section
        if (plugin.screenshots && plugin.screenshots.length > 0) {
            html += `
                <div class="card mb-4">
                    <div class="card-header bg-light">
                        <h6 class="mb-0"><i class="bi bi-images me-2"></i>Screenshots</h6>
                    </div>
                    <div class="card-body">
                        <div class="row">
                            ${plugin.screenshots.map((screenshot, index) => `
                                <div class="col-md-6 mb-3">
                                    <img src="${screenshot}" class="img-fluid rounded shadow-sm" 
                                         alt="Screenshot ${index + 1}" 
                                         style="cursor: pointer;"
                                         onclick="window.open('${screenshot}', '_blank')">
                                </div>
                            `).join('')}
                        </div>
                    </div>
                </div>
            `;
        }
        
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
        if (readmeBtn) {
            readmeBtn.href = `${plugin.repository}/blob/main/README.md`;
        }
        
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
    },
    
    // Handle plugin selection checkbox
    handlePluginSelection: function(checkbox) {
        const pluginId = checkbox.value;
        const repository = checkbox.getAttribute('data-repository');
        
        if (checkbox.checked) {
            this.selectedPlugins.add({id: pluginId, repository: repository});
        } else {
            // Remove from selected plugins
            this.selectedPlugins.forEach(plugin => {
                if (plugin.id === pluginId) {
                    this.selectedPlugins.delete(plugin);
                }
            });
        }
        
        this.updateBulkControls();
    },
    
    // Update bulk controls visibility and count
    updateBulkControls: function() {
        const bulkControls = document.getElementById('bulkActionControls');
        const selectedCount = document.getElementById('selectedCount');
        
        if (this.selectedPlugins.size > 0) {
            bulkControls.style.display = 'block';
            selectedCount.textContent = this.selectedPlugins.size;
        } else {
            bulkControls.style.display = 'none';
        }
    },
    
    // Select all available (non-installed) plugins
    selectAllPlugins: function() {
        const checkboxes = document.querySelectorAll('.plugin-select-checkbox');
        checkboxes.forEach(checkbox => {
            if (!checkbox.checked) {
                checkbox.checked = true;
                this.handlePluginSelection(checkbox);
            }
        });
    },
    
    // Deselect all plugins
    deselectAllPlugins: function() {
        const checkboxes = document.querySelectorAll('.plugin-select-checkbox');
        checkboxes.forEach(checkbox => {
            if (checkbox.checked) {
                checkbox.checked = false;
                this.handlePluginSelection(checkbox);
            }
        });
    },
    
    // Bulk install selected plugins
    bulkInstallPlugins: function() {
        if (this.selectedPlugins.size === 0) {
            this.showToast('Warning', 'No plugins selected for installation', 'warning');
            return;
        }
        
        const selectedArray = Array.from(this.selectedPlugins);
        const repositories = selectedArray.map(plugin => plugin.repository);
        
        // Show confirmation
        this.confirmAction(
            `Are you sure you want to install ${this.selectedPlugins.size} selected plugins?`,
            () => {
                this.showBulkInstallModal(selectedArray);
                this.performBulkInstall(repositories);
            }
        );
    },
    
    // Show bulk install progress modal
    showBulkInstallModal: function(plugins) {
        // Initialize modal
        const modal = new bootstrap.Modal(document.getElementById('bulkInstallModal'));
        
        // Set up progress tracking
        const overallProgress = document.getElementById('overallProgress');
        const overallProgressBar = document.getElementById('overallProgressBar');
        const currentlyInstalling = document.getElementById('currentlyInstalling');
        const installDetailsTable = document.getElementById('installDetailsTable');
        
        // Initialize progress
        overallProgress.textContent = `0 / ${plugins.length}`;
        overallProgressBar.style.width = '0%';
        currentlyInstalling.textContent = 'Preparing installation...';
        
        // Initialize details table
        installDetailsTable.innerHTML = plugins.map(plugin => `
            <tr id="install-row-${plugin.id}">
                <td>${plugin.id}</td>
                <td><span class="badge bg-secondary">Waiting</span></td>
                <td><span id="install-details-${plugin.id}">-</span></td>
            </tr>
        `).join('');
        
        // Show modal
        modal.show();
        
        // Store modal reference
        this.bulkInstallModal = modal;
    },
    
    // Perform bulk installation
    performBulkInstall: function(repositories) {
        const cancelBtn = document.getElementById('cancelInstallBtn');
        const closeBtn = document.getElementById('closeInstallBtn');
        let installationCancelled = false;
        
        // Handle cancellation
        cancelBtn.addEventListener('click', () => {
            installationCancelled = true;
            this.showToast('Info', 'Installation cancelled', 'info');
            this.bulkInstallModal.hide();
        });
        
        // Call bulk install API
        fetch('/api/plugins/manage/bulk-install', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                repositories: repositories
            })
        })
        .then(response => {
            if (!response.ok) {
                throw new Error('Failed to start bulk installation');
            }
            return response.json();
        })
        .then(data => {
            if (installationCancelled) return;
            
            // Update progress based on results
            this.updateBulkInstallProgress(data);
            
            // Switch to close button
            cancelBtn.classList.add('d-none');
            closeBtn.classList.remove('d-none');
            
            // Show completion message
            const overallSuccess = data.overallSuccess;
            const message = overallSuccess 
                ? `Successfully installed ${data.successCount} plugins!`
                : `Installation completed with ${data.failureCount} failures. ${data.successCount} plugins installed successfully.`;
            
            this.showToast(
                overallSuccess ? 'Success' : 'Warning', 
                message, 
                overallSuccess ? 'success' : 'warning'
            );
            
            // Clear selections and refresh
            this.deselectAllPlugins();
            this.loadAvailablePlugins();
            this.loadInstalledPlugins();
        })
        .catch(error => {
            if (installationCancelled) return;
            
            console.error('Error during bulk installation:', error);
            this.showToast('Error', 'Failed to install plugins: ' + error.message, 'error');
            
            // Switch to close button
            cancelBtn.classList.add('d-none');
            closeBtn.classList.remove('d-none');
        });
    },
    
    // Update bulk install progress display
    updateBulkInstallProgress: function(data) {
        const overallProgress = document.getElementById('overallProgress');
        const overallProgressBar = document.getElementById('overallProgressBar');
        const currentlyInstalling = document.getElementById('currentlyInstalling');
        
        // Update overall progress
        const completedCount = data.successCount + data.failureCount;
        const progressPercent = (completedCount / data.totalPlugins) * 100;
        
        overallProgress.textContent = `${completedCount} / ${data.totalPlugins}`;
        overallProgressBar.style.width = `${progressPercent}%`;
        overallProgressBar.classList.remove('progress-bar-striped', 'progress-bar-animated');
        
        if (completedCount === data.totalPlugins) {
            currentlyInstalling.textContent = 'Installation completed';
            overallProgressBar.classList.add('bg-success');
        }
        
        // Update individual plugin status
        data.results.forEach(result => {
            const row = document.getElementById(`install-row-${result.pluginId}`);
            const statusCell = row.querySelector('td:nth-child(2)');
            const detailsCell = document.getElementById(`install-details-${result.pluginId}`);
            
            if (result.success) {
                statusCell.innerHTML = '<span class="badge bg-success">Installed</span>';
                detailsCell.textContent = 'Successfully installed';
            } else {
                statusCell.innerHTML = '<span class="badge bg-danger">Failed</span>';
                detailsCell.textContent = result.error || 'Installation failed';
            }
        });
    }
});

// Add store initialization to the main init function
const originalInit = PluginManagerUI.init;
PluginManagerUI.init = function() {
    originalInit.call(this);
    this.initStore();
};
