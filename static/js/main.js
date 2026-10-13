/* The pages are complete without JavaScript. These are navigation enhancements. */
(() => {
    'use strict';

    const indicator = document.querySelector('.loading-indicator');

    function beginNavigation(message) {
        document.body.classList.add('is-loading');
        if (indicator) {
            indicator.textContent = message || '正在打开页面…';
            indicator.hidden = false;
        }
    }

    function resetNavigation() {
        document.body.classList.remove('is-loading');
        if (indicator) indicator.hidden = true;
        document.querySelectorAll('[aria-busy="true"]').forEach(element => {
            element.removeAttribute('aria-busy');
            if (element.dataset.originalLabel) {
                element.textContent = element.dataset.originalLabel;
                delete element.dataset.originalLabel;
            }
        });
    }

    document.addEventListener('click', event => {
        if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
        const link = event.target.closest('a[href]');
        if (!link || link.hasAttribute('download') || (link.target && link.target !== '_self')) return;
        const destination = new URL(link.href, window.location.href);
        if (destination.origin !== window.location.origin) return;
        if (destination.pathname === window.location.pathname && destination.search === window.location.search && destination.hash) return;
        beginNavigation();
    });

    document.querySelectorAll('[data-loading-form]').forEach(form => {
        form.addEventListener('submit', event => {
            if (event.defaultPrevented) return;
            form.setAttribute('aria-busy', 'true');
            form.querySelectorAll('[data-loading-label]').forEach(button => {
                button.dataset.originalLabel = button.textContent;
                button.textContent = button.dataset.loadingLabel;
                button.setAttribute('aria-busy', 'true');
            });
            beginNavigation('正在查找文章…');
        });
    });

    document.querySelectorAll('[data-retry]').forEach(button => {
        button.hidden = false;
        button.addEventListener('click', () => {
            button.dataset.originalLabel = button.textContent;
            button.textContent = '正在重试…';
            button.setAttribute('aria-busy', 'true');
            beginNavigation('正在重新加载页面…');
            window.location.reload();
        });
    });

    // Browsers may restore this DOM from their back/forward cache.
    window.addEventListener('pageshow', resetNavigation);
})();
