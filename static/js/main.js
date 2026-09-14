document.addEventListener('DOMContentLoaded', () => {
    const postsContainer = document.getElementById('posts');

    loadPosts();

    async function loadPosts() {
        try {
            const response = await fetch('/posts');
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}`);
            }

            const posts = await response.json();
            if (posts.length === 0) {
                postsContainer.replaceChildren(createEmptyState());
                return;
            }

            const fragment = document.createDocumentFragment();
            posts.forEach(post => fragment.appendChild(createPostCard(post)));
            postsContainer.replaceChildren(fragment);
        } catch (error) {
            console.error('Failed to load posts:', error);
            postsContainer.replaceChildren(createMessage('加载文章失败，请刷新页面重试', 'error-state'));
        }
    }

    function createPostCard(post) {
        const link = document.createElement('a');
        link.className = 'post-card-link';
        link.href = `/post/${encodeURIComponent(post.id)}`;

        const article = document.createElement('article');
        article.className = 'post-card';

        const title = document.createElement('h2');
        title.textContent = post.title;

        const excerpt = document.createElement('p');
        excerpt.className = 'post-excerpt';
        excerpt.textContent = post.excerpt;

        const readMore = document.createElement('span');
        readMore.className = 'read-more';
        readMore.textContent = 'Read more';

        article.append(title, excerpt, readMore);
        link.appendChild(article);
        return link;
    }

    function createEmptyState() {
        return createMessage('暂无文章', 'empty-state');
    }

    function createMessage(text, className) {
        const state = document.createElement('div');
        state.className = className;

        const message = document.createElement('p');
        message.textContent = text;
        state.appendChild(message);
        return state;
    }
});
