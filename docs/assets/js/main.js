// Hector Documentation JavaScript

document.addEventListener('DOMContentLoaded', function() {
    // Mobile menu toggle
    const mobileMenuToggle = document.querySelector('.mobile-menu-toggle');
    const mainNav = document.querySelector('.main-nav');
    
    if (mobileMenuToggle && mainNav) {
        mobileMenuToggle.addEventListener('click', function() {
            mainNav.classList.toggle('mobile-open');
            this.classList.toggle('active');
        });
    }
    
    // Smooth scrolling for anchor links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function(e) {
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                e.preventDefault();
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        });
    });
    
    // Copy code blocks
    document.querySelectorAll('pre code').forEach(block => {
        const button = document.createElement('button');
        button.className = 'copy-button';
        button.textContent = 'Copy';
        button.setAttribute('aria-label', 'Copy code to clipboard');
        
        button.addEventListener('click', function() {
            navigator.clipboard.writeText(block.textContent).then(() => {
                button.textContent = 'Copied!';
                button.classList.add('copied');
                setTimeout(() => {
                    button.textContent = 'Copy';
                    button.classList.remove('copied');
                }, 2000);
            });
        });
        
        const pre = block.parentElement;
        pre.style.position = 'relative';
        pre.appendChild(button);
    });
    
    // Feedback buttons
    document.querySelectorAll('.feedback-btn').forEach(button => {
        button.addEventListener('click', function() {
            const feedback = this.dataset.feedback;
            const container = this.parentElement;
            
            // Send feedback (you can implement analytics here)
            console.log('Page feedback:', feedback);
            
            // Update UI
            container.innerHTML = '<p>Thank you for your feedback!</p>';
        });
    });
    
    // Search functionality (basic)
    const searchInput = document.querySelector('.search-input');
    if (searchInput) {
        searchInput.addEventListener('input', function() {
            const query = this.value.toLowerCase();
            const searchResults = document.querySelector('.search-results');
            
            if (query.length > 2) {
                // Simple search implementation
                // In a real implementation, you'd use a proper search index
                performSearch(query);
            } else {
                if (searchResults) {
                    searchResults.style.display = 'none';
                }
            }
        });
    }
    
    // Table of contents generation
    generateTableOfContents();
    
    // Highlight current page in sidebar
    highlightCurrentPage();
    
    // Add external link indicators
    addExternalLinkIndicators();
    
    // Initialize tooltips
    initializeTooltips();
});

function performSearch(query) {
    // Basic search implementation
    // In production, you'd want to use a proper search service like Algolia
    const searchableElements = document.querySelectorAll('h1, h2, h3, h4, p, li');
    const results = [];
    
    searchableElements.forEach(element => {
        if (element.textContent.toLowerCase().includes(query)) {
            results.push({
                title: element.textContent.substring(0, 100),
                url: window.location.pathname + '#' + (element.id || ''),
                context: element.textContent.substring(0, 200)
            });
        }
    });
    
    displaySearchResults(results);
}

function displaySearchResults(results) {
    const searchResults = document.querySelector('.search-results');
    if (!searchResults) return;
    
    if (results.length === 0) {
        searchResults.innerHTML = '<p>No results found.</p>';
    } else {
        const resultsList = results.map(result => `
            <div class="search-result">
                <a href="${result.url}">${result.title}</a>
                <p>${result.context}...</p>
            </div>
        `).join('');
        
        searchResults.innerHTML = resultsList;
    }
    
    searchResults.style.display = 'block';
}

function generateTableOfContents() {
    const tocContainer = document.querySelector('.table-of-contents');
    if (!tocContainer) return;
    
    const headings = document.querySelectorAll('.page-body h2, .page-body h3, .page-body h4');
    if (headings.length === 0) return;
    
    const tocList = document.createElement('ul');
    tocList.className = 'toc-list';
    
    headings.forEach((heading, index) => {
        // Add ID if not present
        if (!heading.id) {
            heading.id = 'heading-' + index;
        }
        
        const listItem = document.createElement('li');
        listItem.className = 'toc-item toc-' + heading.tagName.toLowerCase();
        
        const link = document.createElement('a');
        link.href = '#' + heading.id;
        link.textContent = heading.textContent;
        
        listItem.appendChild(link);
        tocList.appendChild(listItem);
    });
    
    tocContainer.appendChild(tocList);
}

function highlightCurrentPage() {
    const currentPath = window.location.pathname;
    const sidebarLinks = document.querySelectorAll('.sidebar a');
    
    sidebarLinks.forEach(link => {
        if (link.getAttribute('href') === currentPath || 
            link.getAttribute('href') === currentPath.replace('/index', '')) {
            link.classList.add('current');
            
            // Expand parent section if needed
            const section = link.closest('.sidebar-section');
            if (section) {
                section.classList.add('expanded');
            }
        }
    });
}

function addExternalLinkIndicators() {
    const externalLinks = document.querySelectorAll('a[href^="http"]:not([href*="' + window.location.hostname + '"])');
    
    externalLinks.forEach(link => {
        link.classList.add('external-link');
        link.setAttribute('target', '_blank');
        link.setAttribute('rel', 'noopener noreferrer');
        
        // Add icon
        const icon = document.createElement('span');
        icon.className = 'external-icon';
        icon.innerHTML = '↗';
        link.appendChild(icon);
    });
}

function initializeTooltips() {
    const tooltipElements = document.querySelectorAll('[data-tooltip]');
    
    tooltipElements.forEach(element => {
        element.addEventListener('mouseenter', function() {
            const tooltip = document.createElement('div');
            tooltip.className = 'tooltip';
            tooltip.textContent = this.dataset.tooltip;
            document.body.appendChild(tooltip);
            
            const rect = this.getBoundingClientRect();
            tooltip.style.left = rect.left + (rect.width / 2) - (tooltip.offsetWidth / 2) + 'px';
            tooltip.style.top = rect.top - tooltip.offsetHeight - 10 + 'px';
            
            this._tooltip = tooltip;
        });
        
        element.addEventListener('mouseleave', function() {
            if (this._tooltip) {
                document.body.removeChild(this._tooltip);
                this._tooltip = null;
            }
        });
    });
}

// Utility functions
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

function throttle(func, limit) {
    let inThrottle;
    return function() {
        const args = arguments;
        const context = this;
        if (!inThrottle) {
            func.apply(context, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

// Analytics helper (if you want to add tracking)
function trackEvent(category, action, label) {
    if (typeof gtag !== 'undefined') {
        gtag('event', action, {
            event_category: category,
            event_label: label,
        });
    }
}

// Performance monitoring
if ('performance' in window) {
    window.addEventListener('load', function() {
        const loadTime = performance.timing.loadEventEnd - performance.timing.navigationStart;
        console.log('Page load time:', loadTime + 'ms');
        
        // Track performance if analytics is enabled
        trackEvent('Performance', 'Page Load Time', Math.round(loadTime / 100) * 100);
    });
}
