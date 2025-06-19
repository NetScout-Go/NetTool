document.addEventListener('DOMContentLoaded', function() {
    // Mobile menu toggle
    const mobileMenuBtn = document.querySelector('.mobile-menu-btn');
    const navLinks = document.querySelector('.nav-links');
    
    if (mobileMenuBtn) {
        mobileMenuBtn.addEventListener('click', function() {
            navLinks.classList.toggle('show');
            mobileMenuBtn.classList.toggle('active');
        });
    }    // Enhanced theme toggle functionality
    const themeToggle = document.getElementById('theme-toggle');
    if (themeToggle) {        themeToggle.addEventListener('click', function() {
            // Add transition class for animation
            document.body.classList.add('theme-transition');
            
            // Toggle dark mode class on body
            document.body.classList.toggle('dark-mode');
            
            // For dashboard pages, also toggle dashboard-dark-mode class
            if (document.body.classList.contains('dashboard-body')) {
                document.body.classList.toggle('dashboard-dark-mode');
            }
            
            // Apply dark mode to html element as well for complete coverage
            document.documentElement.classList.toggle('dark-mode');
            
            // Update icon visibility
            themeToggle.classList.toggle('dark-active');
            
            // Save preference
            if (document.body.classList.contains('dark-mode')) {
                localStorage.setItem('theme', 'dark');
            } else {
                localStorage.setItem('theme', 'light');
            }
            
            // Dispatch a custom event for other scripts to listen for
            const themeChangeEvent = new CustomEvent('themechange', {
                detail: { theme: document.body.classList.contains('dark-mode') ? 'dark' : 'light' }
            });
            document.dispatchEvent(themeChangeEvent);
            
            // Remove transition class after animation completes
            setTimeout(() => {
                document.body.classList.remove('theme-transition');
            }, 500);
        });
          // Check for saved theme preference and apply to all pages
        if (localStorage.getItem('theme') === 'dark') {
            document.body.classList.add('dark-mode');
            document.documentElement.classList.add('dark-mode');
            themeToggle.classList.add('dark-active');
            
            // Add dashboard dark mode if on dashboard page
            if (document.body.classList.contains('dashboard-body')) {
                document.body.classList.add('dashboard-dark-mode');
            }
            
            // Dispatch initial theme event
            const themeChangeEvent = new CustomEvent('themechange', {
                detail: { theme: 'dark' }
            });
            document.dispatchEvent(themeChangeEvent);
        } else {
            // Dispatch initial theme event
            const themeChangeEvent = new CustomEvent('themechange', {
                detail: { theme: 'light' }
            });
            document.dispatchEvent(themeChangeEvent);
        }
        
        // Listen for system theme preference changes
        if (window.matchMedia) {
            const prefersDarkScheme = window.matchMedia('(prefers-color-scheme: dark)');
            
            // Initial check for system preference if no saved preference
            if (!localStorage.getItem('theme')) {                if (prefersDarkScheme.matches) {
                    document.body.classList.add('dark-mode');
                    document.documentElement.classList.add('dark-mode');
                    themeToggle.classList.add('dark-active');
                    
                    if (document.body.classList.contains('dashboard-body')) {
                        document.body.classList.add('dashboard-dark-mode');
                    }
                    
                    localStorage.setItem('theme', 'dark');
                    
                    // Dispatch theme event
                    const themeChangeEvent = new CustomEvent('themechange', {
                        detail: { theme: 'dark' }
                    });
                    document.dispatchEvent(themeChangeEvent);
                }
            }
            
            // Listen for changes in system preference
            prefersDarkScheme.addEventListener('change', (e) => {
                // Only apply system preference if user hasn't set a preference
                if (!localStorage.getItem('theme')) {                    if (e.matches) {
                        document.body.classList.add('dark-mode');
                        document.documentElement.classList.add('dark-mode');
                        themeToggle.classList.add('dark-active');
                        
                        if (document.body.classList.contains('dashboard-body')) {
                            document.body.classList.add('dashboard-dark-mode');
                        }
                        
                        // Dispatch theme event
                        const themeChangeEvent = new CustomEvent('themechange', {
                            detail: { theme: 'dark' }
                        });
                        document.dispatchEvent(themeChangeEvent);                    } else {
                        document.body.classList.remove('dark-mode');
                        document.documentElement.classList.remove('dark-mode');
                        themeToggle.classList.remove('dark-active');
                        
                        if (document.body.classList.contains('dashboard-body')) {
                            document.body.classList.remove('dashboard-dark-mode');
                        }
                        
                        // Dispatch theme event
                        const themeChangeEvent = new CustomEvent('themechange', {
                            detail: { theme: 'light' }
                        });
                        document.dispatchEvent(themeChangeEvent);
                    }
                }
            });
        }
    }
    
    // Back to top button
    const backToTopButton = document.querySelector('.back-to-top');
    
    if (backToTopButton) {
        // Initially hide the button
        backToTopButton.style.display = 'none';
        
        // Show/hide based on scroll position
        window.addEventListener('scroll', function() {
            if (window.pageYOffset > 300) {
                backToTopButton.style.display = 'flex';
            } else {
                backToTopButton.style.display = 'none';
            }
        });
        
        // Smooth scroll to top when clicked
        backToTopButton.addEventListener('click', function(e) {
            e.preventDefault();
            window.scrollTo({
                top: 0,
                behavior: 'smooth'
            });
        });
    }
    
    // Tabs functionality
    const tabBtns = document.querySelectorAll('.tab-btn');
    const tabPanels = document.querySelectorAll('.tab-panel');
    
    tabBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            // Remove active class from all buttons and panels
            tabBtns.forEach(b => b.classList.remove('active'));
            tabPanels.forEach(p => p.classList.remove('active'));
            
            // Add active class to clicked button
            this.classList.add('active');
            
            // Show corresponding panel
            const category = this.getAttribute('data-category');
            const panel = document.getElementById(category);
            if (panel) {
                panel.classList.add('active');
            }
        });
    });
    
    // Smooth scrolling for navigation links
    const navAnchors = document.querySelectorAll('a[href^="#"]');
    
    navAnchors.forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            const target = document.querySelector(this.getAttribute('href'));
            
            if (target) {
                e.preventDefault();
                
                window.scrollTo({
                    top: target.offsetTop - 80,
                    behavior: 'smooth'
                });
                
                // Close mobile menu if it's open
                if (navLinks.classList.contains('show')) {
                    navLinks.classList.remove('show');
                    mobileMenuBtn.classList.remove('active');
                }
            }
        });
    });
    
    // Copy to clipboard functionality
    const copyBtns = document.querySelectorAll('.copy-btn');
    
    copyBtns.forEach(btn => {
        btn.addEventListener('click', function() {
            const codeBlock = this.closest('.code-block');
            const code = codeBlock.querySelector('code').innerText;
              navigator.clipboard.writeText(code).then(() => {
                // Visual feedback
                const originalIcon = this.innerHTML;
                this.innerHTML = '<i class="fas fa-check"></i>';
                
                // Show notification
                const notification = document.createElement('div');
                notification.className = 'copy-notification';
                notification.textContent = 'Copied to clipboard!';
                document.body.appendChild(notification);
                
                // Set position next to button
                const btnRect = this.getBoundingClientRect();
                notification.style.top = `${btnRect.top - 40}px`;
                notification.style.left = `${btnRect.left - 60}px`;
                
                // Show notification with animation
                setTimeout(() => notification.classList.add('show'), 10);
                
                // Remove notification after delay
                setTimeout(() => {
                    notification.classList.remove('show');
                    setTimeout(() => document.body.removeChild(notification), 300);
                    this.innerHTML = originalIcon;
                }, 2000);
            }).catch(err => {
                console.error('Could not copy text: ', err);
            });
        });
    });
    
    // Form submission
    const newsletterForm = document.querySelector('.newsletter-form');
    
    if (newsletterForm) {
        newsletterForm.addEventListener('submit', function(e) {
            e.preventDefault();
            
            const emailInput = this.querySelector('input[type="email"]');
            const email = emailInput.value;
            
            // Here you would typically send this to a server
            console.log('Subscribed with email:', email);
            
            // Show success message
            this.innerHTML = '<p class="success-message">Thanks for subscribing!</p>';
        });
    }
    
    // Scroll reveal animation
    const revealElements = document.querySelectorAll('.feature-card, .tool-card, .step, .hero-image');
    
    function checkReveal() {
        const windowHeight = window.innerHeight;
        const revealPoint = 150;
        
        revealElements.forEach(el => {
            const revealTop = el.getBoundingClientRect().top;
            
            if (revealTop < windowHeight - revealPoint) {
                el.classList.add('revealed');
            }
        });
    }
    
    // Initial check
    window.addEventListener('load', checkReveal);
    window.addEventListener('scroll', checkReveal);
});

// Add CSS class for reveal animation
document.head.insertAdjacentHTML('beforeend', `
<style>
.feature-card, .tool-card, .step, .hero-image {
    opacity: 0;
    transform: translateY(30px);
    transition: opacity 0.6s ease, transform 0.6s ease;
}
.feature-card.revealed, .tool-card.revealed, .step.revealed, .hero-image.revealed {
    opacity: 1;
    transform: translateY(0);
}
.nav-links.show {
    display: flex;
    flex-direction: column;
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: var(--primary-dark);
    padding: 1rem;
    z-index: 100;
}
.mobile-menu-btn.active span:nth-child(1) {
    transform: rotate(45deg) translate(5px, 5px);
}
.mobile-menu-btn.active span:nth-child(2) {
    opacity: 0;
}
.mobile-menu-btn.active span:nth-child(3) {
    transform: rotate(-45deg) translate(7px, -6px);
}
.success-message {
    color: white;
    font-weight: 500;
    padding: 1rem;
    background-color: var(--success-color);
    border-radius: var(--border-radius);
}
</style>
`);
