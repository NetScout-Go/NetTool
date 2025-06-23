/**
 * NetTool Plugin Manager UI
 * Handles UI interactions for the plugin manager page
 */

const PluginManagerUI = {
    // Initialize the plugin manager UI
    init: function() {
        console.log('Plugin Manager UI initialized');
        
        // Load installed plugins
        this.loadInstalledPlugins();
        
        // Set up event listeners
        this.setupEventListeners();
        
        // Initialize Bootstrap tooltips
        const tooltips = document.querySelectorAll('[data-bs-toggle="tooltip"]');
        tooltips.forEach(tooltip => {
            new bootstrap.Tooltip(tooltip);
        });
    },
    
    // Set up event listeners for all interactive elements
    setupEventListeners: function() {
        // Refresh plugins button
        const refreshBtn = document.getElementById('refreshPluginsBtn');
        if (refreshBtn) {
            refreshBtn.addEventListener('click', () => {
                this.loadInstalledPlugins();
            });
        }
        
        // Check for updates button
        const checkUpdatesBtn = document.getElementById('checkUpdatesBtn');
        if (checkUpdatesBtn) {
            checkUpdatesBtn.addEventListener('click', () => {
                this.checkForUpdates();
            });
        }
        
        // Sync with repository button
        const syncRepoBtn = document.getElementById('syncRepoBtn');
        if (syncRepoBtn) {
            syncRepoBtn.addEventListener('click', () => {
                this.syncWithRepository();
            });
        }
        
        // Update all plugins button
        const updateAllPluginsBtn = document.getElementById('updateAllPluginsBtn');
        if (updateAllPluginsBtn) {
            updateAllPluginsBtn.addEventListener('click', () => {
                this.updateAllPlugins();
            });
        }
        
        // Install plugin form
        const installForm = document.getElementById('installPluginForm');
        if (installForm) {
            installForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.installPlugin();
            });
        }
        
        // Upload plugin form
        const uploadForm = document.getElementById('uploadPluginForm');
        if (uploadForm) {
            uploadForm.addEventListener('submit', (e) => {
                e.preventDefault();
                this.uploadPlugin();
            });
        }
        
        // Confirm action button
        const confirmBtn = document.getElementById('confirmActionBtn');
        if (confirmBtn) {
            confirmBtn.addEventListener('click', () => {
                if (this.pendingAction) {
                    this.pendingAction();
                    this.pendingAction = null;
                    const modal = bootstrap.Modal.getInstance(document.getElementById('confirmActionModal'));
                    modal.hide();
                }
            });
        }
    },
    
    // Load the list of installed plugins
    loadInstalledPlugins: function() {
        // Show loading state
        const tableBody = document.querySelector('#installedPluginsTable tbody');
        tableBody.innerHTML = '<tr class="placeholder-row"><td colspan="5" class="text-center">Loading plugins...</td></tr>';
        
        // Fetch plugin data
        fetch('/api/plugins/manage/list')
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to load plugins');
                }
                return response.json();
            })
            .then(data => {
                this.renderPluginTable(data);
                
                // Update repository stats
                const updateCount = data.filter(plugin => plugin.updateAvailable).length;
                const pluginCountEl = document.getElementById('plugin-count');
                const updateCountEl = document.getElementById('update-count');
                if (pluginCountEl) pluginCountEl.textContent = data.length;
                if (updateCountEl) updateCountEl.textContent = updateCount;
            })
            .catch(error => {
                console.error('Error loading plugins:', error);
                tableBody.innerHTML = '<tr class="placeholder-row"><td colspan="5" class="text-center text-danger">' +
                    'Failed to load plugins: ' + error.message + '</td></tr>';
                this.showToast('Error', 'Failed to load plugins: ' + error.message, 'error');
            });
    },
    
    // Render the plugin table with data
    renderPluginTable: function(plugins) {
        const tableBody = document.querySelector('#installedPluginsTable tbody');
        
        if (!plugins || plugins.length === 0) {
            tableBody.innerHTML = '<tr class="placeholder-row"><td colspan="5" class="text-center">No plugins installed</td></tr>';
            
            // Update repository stats
            const pluginCountEl = document.getElementById('plugin-count');
            const updateCountEl = document.getElementById('update-count');
            if (pluginCountEl) pluginCountEl.textContent = '0';
            if (updateCountEl) updateCountEl.textContent = '0';
            return;
        }
        
        tableBody.innerHTML = '';
        
        // Count updates available
        let updateCount = 0;
        
        plugins.forEach(plugin => {
            const row = document.createElement('tr');
            
            // Create status badge
            const statusBadge = this.getStatusBadge(plugin.status);
            
            // Count updates
            if (plugin.updateAvailable) {
                updateCount++;
            }
            
            row.innerHTML = '<td>' +
                '<div class="d-flex align-items-center">' +
                '<i class="bi bi-' + (plugin.icon || 'plugin') + ' me-2"></i>' +
                '<div>' +
                '<div class="fw-bold">' + plugin.name + '</div>' +
                '<div class="small text-muted">' + plugin.id + '</div>' +
                '</div>' +
                '</div>' +
                '</td>' +
                '<td>' +
                '<div class="d-flex flex-column">' +
                '<span>' + (plugin.version || 'N/A') + '</span>' +
                (plugin.gitVersion ? '<span class="small text-muted">Git: ' + plugin.gitVersion + '</span>' : '') +
                (plugin.updateAvailable ? '<span class="badge bg-success mt-1">Update: ' + plugin.latestVersion + '</span>' : '') +
                '</div>' +
                '</td>' +
                '<td>' + (plugin.author || 'Unknown') + '</td>' +
                '<td>' + statusBadge + '</td>' +
                '<td>' +
                '<div class="btn-group btn-group-sm" role="group">' +
                '<button type="button" class="btn btn-outline-primary" data-plugin-id="' + plugin.id + '" data-action="details" ' +
                'data-bs-toggle="tooltip" data-bs-title="View Details">' +
                '<i class="bi bi-info-circle"></i>' +
                '</button>' +
                '<button type="button" class="btn btn-outline-success ' + (plugin.updateAvailable ? '' : 'disabled') + '" ' +
                'data-plugin-id="' + plugin.id + '" data-action="update" ' +
                'data-bs-toggle="tooltip" data-bs-title="Update Plugin">' +
                '<i class="bi bi-arrow-up-circle"></i>' +
                '</button>' +
                '<button type="button" class="btn btn-outline-danger" data-plugin-id="' + plugin.id + '" data-action="uninstall" ' +
                'data-bs-toggle="tooltip" data-bs-title="Uninstall Plugin">' +
                '<i class="bi bi-trash"></i>' +
                '</button>' +
                '</div>' +
                '</td>';
            
            tableBody.appendChild(row);
            
            // Add event listeners to the action buttons
            row.querySelectorAll('[data-action]').forEach(button => {
                button.addEventListener('click', (e) => {
                    const pluginId = button.getAttribute('data-plugin-id');
                    const action = button.getAttribute('data-action');
                    this.handlePluginAction(action, pluginId, plugin);
                });
            });
        });
        
        // Reinitialize tooltips
        const tooltips = document.querySelectorAll('[data-bs-toggle="tooltip"]');
        tooltips.forEach(tooltip => {
            new bootstrap.Tooltip(tooltip);
        });
    },
    
    // Get appropriate status badge HTML
    getStatusBadge: function(status) {
        if (!status) return '<span class="badge bg-secondary">Unknown</span>';
        
        // Convert the status to lowercase for case-insensitive comparison
        const statusLower = status.toLowerCase();
        
        const statusMap = {
            'active': '<span class="badge bg-success">Active</span>',
            'inactive': '<span class="badge bg-secondary">Inactive</span>',
            'error': '<span class="badge bg-danger">Error</span>',
            'disabled': '<span class="badge bg-secondary">Disabled</span>',
            'update-available': '<span class="badge bg-warning">Update Available</span>',
            'pending': '<span class="badge bg-info">Pending</span>',
            'deprecated': '<span class="badge bg-warning">Deprecated</span>'
        };
        
        return statusMap[statusLower] || '<span class="badge bg-secondary">' + status + '</span>';
    },
    
    // Handle plugin action (details, update, uninstall)
    handlePluginAction: function(action, pluginId, plugin) {
        switch(action) {
            case 'details':
                this.showPluginDetails(pluginId, plugin);
                break;
            case 'update':
                this.confirmAction(
                    'Are you sure you want to update the plugin "' + plugin.name + '"?',
                    () => this.updatePlugin(pluginId)
                );
                break;
            case 'uninstall':
                this.confirmAction(
                    'Are you sure you want to uninstall the plugin "' + plugin.name + '"?',
                    () => this.uninstallPlugin(pluginId)
                );
                break;
        }
    },
    
    // Show plugin details in modal
    showPluginDetails: function(pluginId, plugin) {
        // Fetch detailed plugin information
        fetch('/api/plugins/manage/details/' + pluginId)
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to load plugin details');
                }
                return response.json();
            })
            .then(details => {
                const modalTitle = document.getElementById('pluginDetailsModalLabel');
                const modalContent = document.querySelector('.plugin-details-content');
                
                modalTitle.textContent = details.name;
                
                // Format and display plugin details
                let html = '<div class="plugin-details">';
                
                // Plugin Info Card
                html += '<div class="card mb-4">' +
                    '<div class="card-header bg-light">' +
                    '<h6 class="mb-0"><i class="bi bi-info-circle me-2"></i>Plugin Information</h6>' +
                    '</div>' +
                    '<div class="card-body">' +
                    '<div class="row mb-3">' +
                    '<div class="col-md-4 fw-bold">ID:</div>' +
                    '<div class="col-md-8">' + details.id + '</div>' +
                    '</div>' +
                    '<div class="row mb-3">' +
                    '<div class="col-md-4 fw-bold">Description:</div>' +
                    '<div class="col-md-8">' + details.description + '</div>' +
                    '</div>' +
                    '</div>' +
                    '</div>';
                
                // Repository Card
                html += '<div class="card mb-4">' +
                    '<div class="card-header bg-light">' +
                    '<h6 class="mb-0"><i class="bi bi-git me-2"></i>Repository Information</h6>' +
                    '</div>' +
                    '<div class="card-body">' +
                    '<div class="row mb-3">' +
                    '<div class="col-md-4 fw-bold">Version:</div>' +
                    '<div class="col-md-8">' +
                    (details.version || 'N/A') +
                    (details.updateAvailable ?
                    '<span class="badge bg-success ms-2">Update available: ' + details.latestVersion + '</span>' : '') +
                    '</div>' +
                    '</div>' +
                    '<div class="row mb-3">' +
                    '<div class="col-md-4 fw-bold">Author:</div>' +
                    '<div class="col-md-8">' + (details.author || 'Unknown') + '</div>' +
                    '</div>' +
                    '<div class="row mb-3">' +
                    '<div class="col-md-4 fw-bold">License:</div>' +
                    '<div class="col-md-8">' + (details.license || 'Not specified') + '</div>' +
                    '</div>' +
                    '<div class="row mb-3">' +
                    '<div class="col-md-4 fw-bold">Status:</div>' +
                    '<div class="col-md-8">' +
                    this.getStatusBadge(details.status) +
                    '</div>' +
                    '</div>' +
                    '<div class="row mb-3">' +
                    '<div class="col-md-4 fw-bold">Path:</div>' +
                    '<div class="col-md-8"><code>' + (details.path || 'Unknown') + '</code></div>' +
                    '</div>' +
                    '</div>' +
                    '</div>';
                
                // Update Alert
                if (details.updateAvailable) {
                    html += '<div class="card mb-4 border-warning">' +
                        '<div class="card-header bg-warning text-dark">' +
                        '<h6 class="mb-0"><i class="bi bi-exclamation-triangle me-2"></i>Update Available</h6>' +
                        '</div>' +
                        '<div class="card-body">' +
                        '<p>A newer version is available: <strong>' + (details.latestVersion || 'newer version') + '</strong></p>' +
                        '<button class="btn btn-warning btn-sm" onclick="PluginManagerUI.updatePlugin(\'' + details.id + '\')">' +
                        '<i class="bi bi-arrow-up-circle me-1"></i>Update Now' +
                        '</button>' +
                        '</div>' +
                        '</div>';
                }
                
                // Dependencies Card
                if (details.dependencies && details.dependencies.length > 0) {
                    html += '<div class="card mb-4">' +
                        '<div class="card-header bg-light">' +
                        '<h6 class="mb-0"><i class="bi bi-box me-2"></i>Dependencies</h6>' +
                        '</div>' +
                        '<div class="card-body">' +
                        '<ul class="list-group">';
                    
                    details.dependencies.forEach(dep => {
                        html += '<li class="list-group-item d-flex justify-content-between align-items-center">' +
                            dep.name +
                            '<span class="badge bg-primary rounded-pill">' + (dep.version || 'latest') + '</span>' +
                            '</li>';
                    });
                    
                    html += '</ul>' +
                        '</div>' +
                        '</div>';
                }
                
                html += '</div>'; // End plugin-details
                
                modalContent.innerHTML = html;
                
                // Set up action buttons
                const updateBtn = document.getElementById('updatePluginBtn');
                const uninstallBtn = document.getElementById('uninstallPluginBtn');
                const readmeLink = document.getElementById('viewPluginReadme');
                
                // Update button
                if (updateBtn) {
                    if (details.updateAvailable) {
                        updateBtn.classList.remove('btn-outline-primary');
                        updateBtn.classList.add('btn-primary');
                        updateBtn.disabled = false;
                        updateBtn.innerHTML = '<i class="bi bi-arrow-up-circle"></i> Update to ' + (details.latestVersion || 'latest');
                    } else {
                        updateBtn.disabled = true;
                        updateBtn.innerHTML = '<i class="bi bi-check-circle"></i> Up to date';
                    }
                    
                    updateBtn.onclick = () => {
                        const modal = bootstrap.Modal.getInstance(document.getElementById('pluginDetailsModal'));
                        modal.hide();
                        this.confirmAction(
                            'Are you sure you want to update the plugin "' + details.name + '"?',
                            () => this.updatePlugin(details.id)
                        );
                    };
                }
                
                // Uninstall button
                if (uninstallBtn) {
                    uninstallBtn.onclick = () => {
                        const modal = bootstrap.Modal.getInstance(document.getElementById('pluginDetailsModal'));
                        modal.hide();
                        this.confirmAction(
                            'Are you sure you want to uninstall the plugin "' + details.name + '"?',
                            () => this.uninstallPlugin(details.id)
                        );
                    };
                }
                
                // README link
                if (readmeLink) {
                    const readmePath = details.path + '/README.md';
                    // Check if README exists at this path
                    this.checkFileExists(readmePath)
                        .then(exists => {
                            if (exists) {
                                readmeLink.href = '/api/plugins/manage/view-file?path=' + encodeURIComponent(readmePath);
                                readmeLink.classList.remove('disabled');
                            } else {
                                readmeLink.classList.add('disabled');
                                readmeLink.href = '#';
                                readmeLink.onclick = (e) => {
                                    e.preventDefault();
                                    this.showToast('Info', 'README file not found for this plugin', 'info');
                                };
                            }
                        })
                        .catch(() => {
                            readmeLink.classList.add('disabled');
                            readmeLink.href = '#';
                        });
                }
                
                // Show the modal
                const modal = new bootstrap.Modal(document.getElementById('pluginDetailsModal'));
                modal.show();
            })
            .catch(error => {
                console.error('Error loading plugin details:', error);
                this.showToast('Error', 'Failed to load plugin details: ' + error.message, 'error');
            });
    },
    
    // Check for updates
    checkForUpdates: function() {
        // Show the version management modal
        const modal = new bootstrap.Modal(document.getElementById('versionManagementModal'));
        if (modal) {
            modal.show();
            
            // Show loading state
            const tableBody = document.querySelector('#pluginVersionsTable tbody');
            if (tableBody) {
                tableBody.innerHTML = '<tr class="placeholder-row"><td colspan="5" class="text-center">Checking for updates...</td></tr>';
            }
            
            // Fetch plugin data
            fetch('/api/plugins/manage/list')
                .then(response => {
                    if (!response.ok) {
                        throw new Error('Failed to load plugins');
                    }
                    return response.json();
                })
                .then(plugins => {
                    this.renderVersionTable(plugins);
                })
                .catch(error => {
                    console.error('Error checking for updates:', error);
                    if (tableBody) {
                        tableBody.innerHTML = '<tr class="placeholder-row"><td colspan="5" class="text-center text-danger">' +
                            'Failed to check for updates: ' + error.message + '</td></tr>';
                    }
                    this.showToast('Error', 'Failed to check for updates: ' + error.message, 'error');
                });
        } else {
            this.showToast('Info', 'Checking for updates...', 'info');
            
            // If modal doesn't exist, just fetch and show toast
            fetch('/api/plugins/manage/check-updates', {
                method: 'POST'
            })
                .then(response => {
                    if (!response.ok) {
                        throw new Error('Failed to check for updates');
                    }
                    return response.json();
                })
                .then(data => {
                    this.showToast('Success', 'Check complete. ' + (data.updateAvailable || 0) + ' plugins have updates available.', 'success');
                    this.loadInstalledPlugins();
                })
                .catch(error => {
                    console.error('Error checking for updates:', error);
                    this.showToast('Error', 'Failed to check for updates: ' + error.message, 'error');
                });
        }
    },
    
    // Render the version table
    renderVersionTable: function(plugins) {
        const tableBody = document.querySelector('#pluginVersionsTable tbody');
        if (!tableBody) return;
        
        if (!plugins || plugins.length === 0) {
            tableBody.innerHTML = '<tr class="placeholder-row"><td colspan="5" class="text-center">No plugins installed</td></tr>';
            return;
        }
        
        tableBody.innerHTML = '';
        
        plugins.forEach(plugin => {
            const row = document.createElement('tr');
            
            // Create status badge
            let statusBadge = '';
            if (plugin.updateAvailable) {
                statusBadge = '<span class="badge bg-warning">Update Available</span>';
            } else {
                statusBadge = '<span class="badge bg-success">Up to Date</span>';
            }
            
            row.innerHTML = '<td>' +
                '<div class="d-flex align-items-center">' +
                '<i class="bi bi-' + (plugin.icon || 'plugin') + ' me-2"></i>' +
                '<div>' +
                '<div class="fw-bold">' + plugin.name + '</div>' +
                '<div class="small text-muted">' + plugin.id + '</div>' +
                '</div>' +
                '</div>' +
                '</td>' +
                '<td>' + (plugin.version || 'N/A') + '</td>' +
                '<td>' + (plugin.latestVersion || plugin.version || 'N/A') + '</td>' +
                '<td>' + statusBadge + '</td>' +
                '<td>' +
                '<button type="button" class="btn btn-sm btn-outline-primary ' + (!plugin.updateAvailable ? 'disabled' : '') + '" ' +
                'data-plugin-id="' + plugin.id + '" data-action="update-version">' +
                '<i class="bi bi-arrow-up-circle"></i> Update' +
                '</button>' +
                '</td>';
            
            tableBody.appendChild(row);
            
            // Add event listener to update button
            const updateBtn = row.querySelector('[data-action="update-version"]');
            if (updateBtn && !updateBtn.classList.contains('disabled')) {
                updateBtn.addEventListener('click', () => {
                    this.updatePlugin(plugin.id);
                });
            }
        });
    },
    
    // Sync with repository
    syncWithRepository: function() {
        // Show loading state
        const syncBtn = document.getElementById('syncRepoBtn');
        if (!syncBtn) return;
        
        const originalBtnText = syncBtn.innerHTML;
        syncBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Syncing...';
        syncBtn.disabled = true;
        
        // Fetch from repository
        fetch('/api/plugins/manage/sync', {
            method: 'POST'
        })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to sync with repository');
                }
                return response.json();
            })
            .then(data => {
                this.showToast('Success', 'Successfully synced with repository. ' + (data.updated || 0) + ' plugins updated.', 'success');
                this.loadInstalledPlugins();
            })
            .catch(error => {
                console.error('Error syncing with repository:', error);
                this.showToast('Error', 'Failed to sync with repository: ' + error.message, 'error');
            })
            .finally(() => {
                syncBtn.innerHTML = originalBtnText;
                syncBtn.disabled = false;
            });
    },
    
    // Update all plugins
    updateAllPlugins: function() {
        // Show confirmation
        this.confirmAction('Are you sure you want to update all plugins? This may take a while.', () => {
            // Show loading state
            const updateBtn = document.getElementById('updateAllPluginsBtn');
            if (!updateBtn) return;
            
            const originalBtnText = updateBtn.innerHTML;
            updateBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Updating...';
            updateBtn.disabled = true;
            
            // Update all plugins
            fetch('/api/plugins/manage/update-all', {
                method: 'POST'
            })
                .then(response => {
                    if (!response.ok) {
                        throw new Error('Failed to update plugins');
                    }
                    return response.json();
                })
                .then(data => {
                    this.showToast('Success', 'Successfully updated ' + (data.updated || 0) + ' plugins.', 'success');
                    this.loadInstalledPlugins();
                    
                    // Refresh the version modal if it's open
                    const versionModal = document.getElementById('versionManagementModal');
                    if (versionModal && versionModal.classList.contains('show')) {
                        this.checkForUpdates();
                    }
                })
                .catch(error => {
                    console.error('Error updating plugins:', error);
                    this.showToast('Error', 'Failed to update plugins: ' + error.message, 'error');
                })
                .finally(() => {
                    updateBtn.innerHTML = originalBtnText;
                    updateBtn.disabled = false;
                });
        });
    },
    
    // Install a plugin from URL
    installPlugin: function() {
        const pluginUrl = document.getElementById('pluginUrl');
        if (!pluginUrl) return;
        
        const url = pluginUrl.value.trim();
        if (!url) {
            this.showToast('Error', 'Please enter a valid plugin URL', 'error');
            return;
        }
        
        // Show loading state
        const installBtn = document.getElementById('installPluginBtn');
        if (!installBtn) return;
        
        const originalBtnText = installBtn.innerHTML;
        installBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Installing...';
        installBtn.disabled = true;
        
        // Submit the installation request
        fetch('/api/plugins/manage/install', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ url: url })
        })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to install plugin');
                }
                return response.json();
            })
            .then(data => {
                this.showToast('Success', 'Plugin "' + data.name + '" installed successfully', 'success');
                pluginUrl.value = '';
                this.loadInstalledPlugins();
            })
            .catch(error => {
                console.error('Error installing plugin:', error);
                this.showToast('Error', 'Failed to install plugin: ' + error.message, 'error');
            })
            .finally(() => {
                // Restore button state
                installBtn.innerHTML = originalBtnText;
                installBtn.disabled = false;
            });
    },
    
    // Upload a plugin zip file
    uploadPlugin: function() {
        const fileInput = document.getElementById('pluginFile');
        if (!fileInput) return;
        
        const file = fileInput.files[0];
        if (!file) {
            this.showToast('Error', 'Please select a plugin file', 'error');
            return;
        }
        
        // Show loading state
        const uploadBtn = document.getElementById('uploadPluginBtn');
        if (!uploadBtn) return;
        
        const originalBtnText = uploadBtn.innerHTML;
        uploadBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span> Uploading...';
        uploadBtn.disabled = true;
        
        // Create form data
        const formData = new FormData();
        formData.append('plugin', file);
        
        // Submit the upload request
        fetch('/api/plugins/manage/upload', {
            method: 'POST',
            body: formData
        })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to upload plugin');
                }
                return response.json();
            })
            .then(data => {
                this.showToast('Success', 'Plugin "' + data.name + '" uploaded successfully', 'success');
                fileInput.value = '';
                this.loadInstalledPlugins();
            })
            .catch(error => {
                console.error('Error uploading plugin:', error);
                this.showToast('Error', 'Failed to upload plugin: ' + error.message, 'error');
            })
            .finally(() => {
                // Restore button state
                uploadBtn.innerHTML = originalBtnText;
                uploadBtn.disabled = false;
            });
    },
    
    // Update a plugin
    updatePlugin: function(pluginId) {
        // Show loading state in the table
        const updateBtn = document.querySelector('[data-plugin-id="' + pluginId + '"][data-action="update"]');
        if (updateBtn) {
            updateBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>';
            updateBtn.disabled = true;
        }
        
        // Submit the update request
        fetch('/api/plugins/manage/update/' + pluginId, {
            method: 'POST'
        })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to update plugin');
                }
                return response.json();
            })
            .then(data => {
                this.showToast('Success', 'Plugin "' + data.name + '" updated successfully', 'success');
                this.loadInstalledPlugins();
                
                // Refresh the version modal if it's open
                const versionModal = document.getElementById('versionManagementModal');
                if (versionModal && versionModal.classList.contains('show')) {
                    this.checkForUpdates();
                }
            })
            .catch(error => {
                console.error('Error updating plugin:', error);
                this.showToast('Error', 'Failed to update plugin: ' + error.message, 'error');
                
                // Restore button state if it still exists
                if (updateBtn) {
                    updateBtn.innerHTML = '<i class="bi bi-arrow-up-circle"></i>';
                    updateBtn.disabled = false;
                }
            });
    },
    
    // Uninstall a plugin
    uninstallPlugin: function(pluginId) {
        // Show loading state in the table
        const uninstallBtn = document.querySelector('[data-plugin-id="' + pluginId + '"][data-action="uninstall"]');
        const row = uninstallBtn ? uninstallBtn.closest('tr') : null;
        
        if (row) {
            row.classList.add('table-secondary');
            uninstallBtn.innerHTML = '<span class="spinner-border spinner-border-sm" role="status" aria-hidden="true"></span>';
            uninstallBtn.disabled = true;
        }
        
        // Submit the uninstall request
        fetch('/api/plugins/manage/uninstall/' + pluginId, {
            method: 'POST'
        })
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to uninstall plugin');
                }
                return response.json();
            })
            .then(data => {
                this.showToast('Success', 'Plugin "' + data.name + '" uninstalled successfully', 'success');
                this.loadInstalledPlugins();
            })
            .catch(error => {
                console.error('Error uninstalling plugin:', error);
                this.showToast('Error', 'Failed to uninstall plugin: ' + error.message, 'error');
                
                // Restore row state if it still exists
                if (row) {
                    row.classList.remove('table-secondary');
                    if (uninstallBtn) {
                        uninstallBtn.innerHTML = '<i class="bi bi-trash"></i>';
                        uninstallBtn.disabled = false;
                    }
                }
            });
    },
    
    // Check if a file exists
    checkFileExists: function(path) {
        return fetch('/api/plugins/manage/file-exists?path=' + encodeURIComponent(path))
            .then(response => {
                if (!response.ok) {
                    throw new Error('Failed to check file existence');
                }
                return response.json();
            })
            .then(data => data.exists)
            .catch(() => false);
    },
    
    // Show confirmation modal
    confirmAction: function(message, callback) {
        const confirmMsg = document.getElementById('confirmActionMessage');
        if (!confirmMsg) return;
        
        confirmMsg.textContent = message;
        this.pendingAction = callback;
        
        const modal = new bootstrap.Modal(document.getElementById('confirmActionModal'));
        modal.show();
    },
    
    // Show toast notification
    showToast: function(title, message, type = 'info') {
        const toastEl = document.getElementById('toastNotification');
        const toastTitle = document.getElementById('toastTitle');
        const toastBody = document.getElementById('toastMessage');
        
        if (!toastEl || !toastTitle || !toastBody) return;
        
        // Set type classes
        toastEl.className = 'toast';
        switch(type) {
            case 'success':
                toastEl.classList.add('bg-success', 'text-white');
                break;
            case 'error':
                toastEl.classList.add('bg-danger', 'text-white');
                break;
            case 'warning':
                toastEl.classList.add('bg-warning');
                break;
            default:
                toastEl.classList.add('bg-info', 'text-white');
        }
        
        // Set content
        toastTitle.textContent = title;
        toastBody.textContent = message;
        
        // Show toast
        const toast = new bootstrap.Toast(toastEl);
        toast.show();
    },
    
    // Store pending action for confirmation
    pendingAction: null
};

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    PluginManagerUI.init();
});
