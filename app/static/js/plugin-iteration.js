// Iteration Dialog Component
class IterationDialog {
    constructor() {
        this.modalId = 'continueIterationModal';
        this.createModal();
        this.setupEventListeners();
        this.iterationCallback = null;
    }

    // Create the modal dialog
    createModal() {
        // Check if modal already exists
        if (document.getElementById(this.modalId)) {
            return;
        }

        const modalHtml = `
            <div class="modal fade" id="${this.modalId}" tabindex="-1" aria-labelledby="continueIterationModalLabel" aria-hidden="true">
                <div class="modal-dialog">
                    <div class="modal-content">
                        <div class="modal-header">
                            <h5 class="modal-title" id="continueIterationModalLabel">
                                <i class="bi bi-arrow-repeat text-primary me-2"></i>Continue to iterate?
                            </h5>
                            <button type="button" class="btn-close" data-bs-dismiss="modal" aria-label="Close"></button>
                        </div>
                        <div class="modal-body">
                            <p>The plugin has completed <span id="iterationCount">0</span> iterations.</p>
                            <p>Would you like to continue running this plugin?</p>
                            <div class="progress mb-3">
                                <div id="iterationProgress" class="progress-bar" role="progressbar" style="width: 0%;" aria-valuenow="0" aria-valuemin="0" aria-valuemax="100"></div>
                            </div>
                            <div class="form-check">
                                <input class="form-check-input" type="checkbox" id="dontAskAgain">
                                <label class="form-check-label" for="dontAskAgain">
                                    Don't ask again for this session
                                </label>
                            </div>
                        </div>
                        <div class="modal-footer">
                            <button type="button" class="btn btn-secondary" data-bs-dismiss="modal" id="stopIterationBtn">
                                <i class="bi bi-stop-circle me-1"></i>Stop
                            </button>
                            <button type="button" class="btn btn-primary" id="continueIterationBtn">
                                <i class="bi bi-play-circle me-1"></i>Continue
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        `;

        // Append to body
        const modalContainer = document.createElement('div');
        modalContainer.innerHTML = modalHtml;
        document.body.appendChild(modalContainer);
    }

    // Set up event listeners
    setupEventListeners() {
        // When modal is hidden, call the callback with false
        document.addEventListener('hidden.bs.modal', (event) => {
            if (event.target.id === this.modalId && this.iterationCallback) {
                this.iterationCallback(false);
                this.iterationCallback = null;
            }
        });

        // Stop button
        document.addEventListener('click', (event) => {
            if (event.target.id === 'stopIterationBtn' && this.iterationCallback) {
                this.iterationCallback(false);
                this.iterationCallback = null;
            }
        });

        // Continue button
        document.addEventListener('click', (event) => {
            if (event.target.id === 'continueIterationBtn' && this.iterationCallback) {
                const dontAskAgain = document.getElementById('dontAskAgain').checked;
                this.iterationCallback(true, dontAskAgain);
                this.iterationCallback = null;
            }
        });
    }

    // Show the dialog
    show(iterationCount, maxIterations, callback) {
        this.iterationCallback = callback;
        
        // Update the iteration count
        const countElement = document.getElementById('iterationCount');
        if (countElement) {
            countElement.textContent = iterationCount;
        }
        
        // Update the progress bar
        const progressElement = document.getElementById('iterationProgress');
        if (progressElement && maxIterations > 0) {
            const percentage = Math.min((iterationCount / maxIterations) * 100, 100);
            progressElement.style.width = `${percentage}%`;
            progressElement.setAttribute('aria-valuenow', percentage);
            
            // Update progress bar color
            progressElement.className = 'progress-bar';
            if (percentage < 50) {
                progressElement.classList.add('bg-success');
            } else if (percentage < 75) {
                progressElement.classList.add('bg-info');
            } else if (percentage < 90) {
                progressElement.classList.add('bg-warning');
            } else {
                progressElement.classList.add('bg-danger');
            }
        } else if (progressElement) {
            // Hide progress if maxIterations is 0 (unlimited)
            progressElement.style.width = '100%';
            progressElement.classList.add('bg-info');
        }
        
        // Reset the checkbox
        const checkbox = document.getElementById('dontAskAgain');
        if (checkbox) {
            checkbox.checked = false;
        }
        
        // Show the modal
        const modal = new bootstrap.Modal(document.getElementById(this.modalId));
        modal.show();
    }
}

// Plugin Iteration Manager
class PluginIterationManager {
    constructor(pluginId) {
        this.pluginId = pluginId;
        this.iterating = false;
        this.iterationCount = 0;
        this.maxIterations = 0;
        this.iterationDelay = 5000; // Default 5 seconds
        this.params = {};
        this.results = [];
        this.dontAskAgain = false;
        this.dialog = new IterationDialog();
        this.onIterationResult = null;
        this.onIterationComplete = null;
    }
    
    // Start iteration
    start(params, onResult, onComplete) {
        this.params = params;
        this.iterating = true;
        this.iterationCount = 0;
        this.results = [];
        this.onIterationResult = onResult;
        this.onIterationComplete = onComplete;
        
        // Extract iteration parameters
        this.maxIterations = params.maxIterations || 0;
        this.iterationDelay = params.iterationDelay || 5000;
        
        // Run first iteration
        this.runIteration();
    }
    
    // Stop iteration
    stop() {
        this.iterating = false;
        if (this.onIterationComplete) {
            this.onIterationComplete(this.results);
        }
    }
    
    // Run a single iteration
    runIteration() {
        if (!this.iterating) {
            return;
        }
        
        // Make a copy of params to avoid modifying the original
        const iterationParams = {...this.params, iterationCount: this.iterationCount};
        
        // Run the plugin
        fetch(`/api/plugins/${this.pluginId}/run`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(iterationParams),
        })
        .then(response => response.json())
        .then(result => {
            // Add to results
            this.results.push({
                iterationCount: this.iterationCount,
                result: result,
                timestamp: new Date(),
            });
            
            // Call result callback
            if (this.onIterationResult) {
                this.onIterationResult(result, this.iterationCount);
            }
            
            // Increment iteration count
            this.iterationCount++;
            
            // Check if we should stop based on maxIterations
            if (this.maxIterations > 0 && this.iterationCount >= this.maxIterations) {
                this.stop();
                return;
            }
            
            // Ask if we should continue (every 5 iterations or based on user preference)
            if (!this.dontAskAgain && this.iterationCount > 0 && this.iterationCount % 5 === 0) {
                this.promptToContinue();
            } else {
                // Continue automatically after delay
                setTimeout(() => this.runIteration(), this.iterationDelay);
            }
        })
        .catch(error => {
            console.error('Error running plugin iteration:', error);
            this.stop();
        });
    }
    
    // Prompt to continue iteration
    promptToContinue() {
        this.dialog.show(this.iterationCount, this.maxIterations, (continueIteration, dontAskAgain) => {
            if (continueIteration) {
                this.dontAskAgain = dontAskAgain;
                setTimeout(() => this.runIteration(), this.iterationDelay);
            } else {
                this.stop();
            }
        });
    }
}

// Export the iteration manager
window.PluginIterationManager = PluginIterationManager;
