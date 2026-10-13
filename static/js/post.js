/* Reading enhancements; article content and navigation come from the server. */
(() => {
    'use strict';

    const content = document.querySelector('.post-content');
    if (!content) return;

    async function copyText(value) {
        if (navigator.clipboard && window.isSecureContext) {
            try {
                await navigator.clipboard.writeText(value);
                return;
            } catch (_) {
                // The local HTTP preview can still use the selection fallback.
            }
        }
        const textarea = document.createElement('textarea');
        textarea.value = value;
        textarea.className = 'sr-only';
        textarea.setAttribute('readonly', '');
        document.body.appendChild(textarea);
        const previousFocus = document.activeElement;
        textarea.select();
        let copied = false;
        try {
            copied = document.execCommand('copy');
        } finally {
            textarea.remove();
            if (previousFocus && typeof previousFocus.focus === 'function') previousFocus.focus({ preventScroll: true });
        }
        if (!copied) throw new Error('Clipboard unavailable');
    }

    content.querySelectorAll('pre').forEach(pre => {
        const wrapper = document.createElement('div');
        wrapper.className = 'code-block';
        pre.replaceWith(wrapper);
        wrapper.appendChild(pre);
        pre.tabIndex = 0;
        pre.setAttribute('aria-label', '代码块，可横向滚动');

        const button = document.createElement('button');
        button.className = 'copy-code';
        button.type = 'button';
        button.textContent = '复制代码';
        button.setAttribute('aria-live', 'polite');
        wrapper.appendChild(button);
        let resetTimer;
        button.addEventListener('click', async () => {
            window.clearTimeout(resetTimer);
            const code = pre.querySelector('code') || pre;
            try {
                await copyText(code.textContent);
                button.textContent = '已复制';
                button.dataset.state = 'success';
            } catch (_) {
                button.textContent = '请选中代码复制';
                button.dataset.state = 'error';
            }
            resetTimer = window.setTimeout(() => {
                button.textContent = '复制代码';
                delete button.dataset.state;
            }, 2500);
        });
    });

    const dialog = document.getElementById('image-viewer');
    const expandedImage = document.getElementById('expanded-image');
    const imageCaption = document.getElementById('expanded-image-caption');
    content.querySelectorAll('img').forEach(image => {
        image.loading = 'lazy';
        image.decoding = 'async';
        if (!dialog || typeof dialog.showModal !== 'function' || image.closest('a')) return;
        const button = document.createElement('button');
        button.className = 'image-expand';
        button.type = 'button';
        button.setAttribute('aria-label', image.alt ? `查看大图：${image.alt}` : '查看大图');
        image.replaceWith(button);
        button.appendChild(image);
        button.addEventListener('click', () => {
            expandedImage.src = image.currentSrc || image.src;
            expandedImage.alt = image.alt;
            imageCaption.textContent = image.alt;
            imageCaption.hidden = !image.alt;
            dialog.showModal();
        });
    });
    if (dialog) {
        dialog.querySelector('.image-close').addEventListener('click', () => dialog.close());
        dialog.addEventListener('click', event => {
            if (event.target === dialog) dialog.close();
        });
        dialog.addEventListener('close', () => expandedImage.removeAttribute('src'));
    }

    const tocLinks = Array.from(document.querySelectorAll('[data-toc-link]'));
    const sections = tocLinks.map(link => ({
        link,
        heading: document.getElementById(link.hash.slice(1)),
    })).filter(section => section.heading);
    if (!sections.length) return;

    function selectSection(selected) {
        sections.forEach(section => {
            if (section === selected) section.link.setAttribute('aria-current', 'location');
            else section.link.removeAttribute('aria-current');
        });
    }

    function updateCurrentSection() {
        let selected = sections[0];
        for (const section of sections) {
            if (section.heading.getBoundingClientRect().top <= window.innerHeight * 0.3) selected = section;
        }
        selectSection(selected);
    }

    sections.forEach(section => {
        section.heading.tabIndex = -1;
        section.link.addEventListener('click', () => {
            selectSection(section);
            window.requestAnimationFrame(() => section.heading.focus({ preventScroll: true }));
        });
    });

    if ('IntersectionObserver' in window) {
        const observer = new IntersectionObserver(updateCurrentSection, { rootMargin: '0px 0px -65% 0px', threshold: 0 });
        sections.forEach(section => observer.observe(section.heading));
    }
    window.addEventListener('hashchange', updateCurrentSection);
    updateCurrentSection();
})();
