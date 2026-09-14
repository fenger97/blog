document.addEventListener('DOMContentLoaded', () => {
    const postDetail = document.getElementById('postDetail');
    const pageTitle = document.getElementById('pageTitle');
    const pathname = window.location.pathname.replace(/\/+$/, '');
    const segments = pathname.split('/').filter(Boolean);
    const postID = segments.length >= 2 ? segments[segments.length - 1] : '';

    if (!postID) {
        renderError();
        return;
    }

    loadPost(postID);

    async function loadPost(id) {
        try {
            const response = await fetch(`/api/post?id=${encodeURIComponent(id)}`);
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const post = await response.json();
            pageTitle.textContent = `${post.title} - Fenger's Page`;

            const header = document.createElement('header');
            header.className = 'post-header';

            const title = document.createElement('h1');
            title.className = 'post-title';
            title.textContent = post.title;

            const content = document.createElement('div');
            content.className = 'post-content';
            content.innerHTML = post.html_content;

            header.appendChild(title);
            postDetail.replaceChildren(header, content);
        } catch (error) {
            console.error('Failed to load post:', error);
            renderError();
        }
    }

    function renderError() {
        const state = document.createElement('div');
        state.className = 'error-state';

        const message = document.createElement('p');
        message.textContent = '文章加载失败';

        const link = document.createElement('a');
        link.className = 'back-link';
        link.href = '/';
        link.textContent = '返回首页';

        state.append(message, link);
        postDetail.replaceChildren(state);
    }
});
