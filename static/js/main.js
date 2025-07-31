document.addEventListener('DOMContentLoaded', () => {
    // 按钮
    const logoutBtn = document.getElementById('logoutBtn');
    const newPostBtn = document.getElementById('newPostBtn');
    const cancelBtn = document.getElementById('cancelBtn');

    // 表单
    const postForm = document.getElementById('postForm');
    const createPostForm = document.getElementById('createPostForm');

    // 内容容器
    const postsContainer = document.getElementById('posts');

    // Markdown 编辑器相关
    const tabBtns = document.querySelectorAll('.tab-btn');
    const contentTextarea = document.getElementById('content');
    const previewDiv = document.getElementById('preview');

    // Markdown 编辑器标签页切换
    tabBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.dataset.tab;
            
            // 更新按钮状态
            tabBtns.forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            
            // 切换内容显示
            if (tab === 'edit') {
                contentTextarea.classList.remove('hidden');
                previewDiv.classList.add('hidden');
            } else if (tab === 'preview') {
                contentTextarea.classList.add('hidden');
                previewDiv.classList.remove('hidden');
                
                // 实时预览 Markdown
                updatePreview();
            }
        });
    });

    // 更新预览内容
    function updatePreview() {
        const markdown = contentTextarea.value;
        // 简单的 Markdown 预览（实际项目中可以用更完整的 Markdown 解析库）
        const html = simpleMarkdownToHtml(markdown);
        previewDiv.innerHTML = html;
    }

    // 简单的 Markdown 转 HTML 函数（用于预览）
    function simpleMarkdownToHtml(markdown) {
        return markdown
            .replace(/^### (.*$)/gim, '<h3>$1</h3>')
            .replace(/^## (.*$)/gim, '<h2>$1</h2>')
            .replace(/^# (.*$)/gim, '<h1>$1</h1>')
            .replace(/\*\*(.*)\*\*/gim, '<strong>$1</strong>')
            .replace(/\*(.*)\*/gim, '<em>$1</em>')
            .replace(/`(.*)`/gim, '<code>$1</code>')
            .replace(/\n/gim, '<br>');
    }

    // 检查登录状态并更新UI
    async function checkLoginStatus() {
        try {
            const response = await fetch('/status');
            const data = await response.json();
            if (data.logged_in) {
                logoutBtn.classList.remove('hidden');
                newPostBtn.classList.remove('hidden');
            } else {
                logoutBtn.classList.add('hidden');
                newPostBtn.classList.add('hidden');
            }
        } catch (error) {
            console.error('检查登录状态失败:', error);
        }
    }

    // 退出登录
    logoutBtn.addEventListener('click', async () => {
        try {
            await fetch('/logout', { method: 'POST' });
            checkLoginStatus(); // 更新UI
        } catch (error) {
            console.error('退出登录失败:', error);
        }
    });

    // 显示/隐藏新文章表单
    newPostBtn.addEventListener('click', () => {
        postForm.classList.remove('hidden');
    });

    cancelBtn.addEventListener('click', () => {
        postForm.classList.add('hidden');
        createPostForm.reset();
        // 重置编辑器状态
        tabBtns[0].click(); // 切换到编辑模式
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
            } else if (response.status === 401) {
                alert('请先登录再发布文章');
                checkLoginStatus(); // 会话可能已过期，更新UI
            }
            else {
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
            const posts = await response.json() || [];
            
            postsContainer.innerHTML = posts.map(post => `
                <article class="post-card">
                    <h2>${post.title}</h2>
                    <div class="post-meta">
                        发布于 ${new Date(post.created_at).toLocaleString()}
                    </div>
                    <div class="post-content">
                        ${post.html_content || post.content}
                    </div>
                </article>
            `).join('');
        } catch (error) {
            console.error('Error:', error);
            postsContainer.innerHTML = '<p>加载文章失败，请刷新页面重试</p>';
        }
    }

    // 初始加载
    checkLoginStatus();
    loadPosts();
}); 