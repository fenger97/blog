document.addEventListener('DOMContentLoaded', () => {
    const newPostBtn = document.getElementById('newPostBtn');
    const postForm = document.getElementById('postForm');
    const createPostForm = document.getElementById('createPostForm');
    const cancelBtn = document.getElementById('cancelBtn');
    const postsContainer = document.getElementById('posts');

    // 显示/隐藏新文章表单
    newPostBtn.addEventListener('click', () => {
        postForm.classList.remove('hidden');
    });

    cancelBtn.addEventListener('click', () => {
        postForm.classList.add('hidden');
        createPostForm.reset();
    });

    // 提交新文章
    createPostForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        
        const formData = {
            title: document.getElementById('title').value,
            content: document.getElementById('content').value
        };

        try {
            const response = await fetch('/posts', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(formData)
            });

            if (response.ok) {
                postForm.classList.add('hidden');
                createPostForm.reset();
                loadPosts(); // 重新加载文章列表
            } else {
                alert('发布文章失败，请重试');
            }
        } catch (error) {
            console.error('Error:', error);
            alert('发布文章失败，请重试');
        }
    });

    // 加载文章列表
    async function loadPosts() {
        try {
            const response = await fetch('/posts');
            const posts = await response.json();
            
            postsContainer.innerHTML = posts.map(post => `
                <article class="post-card">
                    <h2>${post.title}</h2>
                    <div class="post-meta">
                        发布于 ${new Date(post.created_at).toLocaleString()}
                    </div>
                    <div class="post-content">
                        ${post.content}
                    </div>
                </article>
            `).join('');
        } catch (error) {
            console.error('Error:', error);
            postsContainer.innerHTML = '<p>加载文章失败，请刷新页面重试</p>';
        }
    }

    // 初始加载文章列表
    loadPosts();
}); 