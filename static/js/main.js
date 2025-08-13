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
    const uploadImageBtn = document.getElementById('uploadImageBtn');
    const imageInput = document.getElementById('imageInput');

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

    // 图片上传功能
    uploadImageBtn.addEventListener('click', () => {
        imageInput.click();
    });

    imageInput.addEventListener('change', async (e) => {
        const file = e.target.files[0];
        if (!file) return;

        // 检查文件类型
        if (!file.type.startsWith('image/')) {
            alert('请选择图片文件');
            return;
        }

        // 检查文件大小 (10MB)
        if (file.size > 10 * 1024 * 1024) {
            alert('图片大小不能超过 10MB');
            return;
        }

        // 创建 FormData
        const formData = new FormData();
        formData.append('image', file);

        try {
            uploadImageBtn.textContent = '上传中...';
            uploadImageBtn.disabled = true;

            const response = await fetch('/upload/image', {
                method: 'POST',
                body: formData
            });

            if (response.ok) {
                const result = await response.json();
                if (result.success) {
                    // 插入 Markdown 图片语法
                    const imageMarkdown = `![图片](${result.url})`;
                    const textarea = document.getElementById('content');
                    const start = textarea.selectionStart;
                    const end = textarea.selectionEnd;
                    const text = textarea.value;
                    
                    textarea.value = text.substring(0, start) + imageMarkdown + text.substring(end);
                    textarea.focus();
                    textarea.setSelectionRange(start + imageMarkdown.length, start + imageMarkdown.length);
                    
                    // 如果当前在预览模式，更新预览
                    if (previewDiv.classList.contains('hidden') === false) {
                        updatePreview();
                    }
                } else {
                    alert('上传失败：' + (result.error || '未知错误'));
                }
            } else {
                alert('上传失败，请重试');
            }
        } catch (error) {
            console.error('Upload error:', error);
            alert('上传失败，请重试');
        } finally {
            uploadImageBtn.textContent = '📷 插入图片';
            uploadImageBtn.disabled = false;
            imageInput.value = ''; // 清空文件选择
        }
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
            .replace(/!\[(.*?)\]\((.*?)\)/gim, '<img src="$2" alt="$1" style="max-width: 100%; height: auto; border-radius: 4px; margin: 1rem 0;">')
            .replace(/\[(.*?)\]\((.*?)\)/gim, '<a href="$2" target="_blank">$1</a>')
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
            
            if (posts.length === 0) {
                postsContainer.innerHTML = `
                    <div class="empty-state">
                        <p>还没有文章，登录后开始写作吧！</p>
                    </div>
                `;
                return;
            }
            
            postsContainer.innerHTML = posts.map(post => `
                <article class="post-card">
                    <h2>
                        <a href="/post/${post.id}" class="post-title-link">${post.title}</a>
                    </h2>
                    <div class="post-excerpt">
                        ${post.excerpt}
                    </div>
                    <div class="post-actions">
                        <a href="/post/${post.id}" class="read-more-btn">Read more</a>
                    </div>
                </article>
            `).join('');
        } catch (error) {
            console.error('Error:', error);
            postsContainer.innerHTML = '<p>加载文章失败，请刷新页面重试</p>';
        }
    }

    // 格式化日期
    function formatDate(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diffTime = Math.abs(now - date);
        const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
        
        if (diffDays === 1) {
            return '昨天';
        } else if (diffDays < 7) {
            return `${diffDays} 天前`;
        } else if (diffDays < 30) {
            const weeks = Math.floor(diffDays / 7);
            return `${weeks} 周前`;
        } else if (diffDays < 365) {
            const months = Math.floor(diffDays / 30);
            return `${months} 个月前`;
        } else {
            return date.toLocaleDateString('zh-CN', {
                year: 'numeric',
                month: 'long',
                day: 'numeric'
            });
        }
    }

    // 初始加载
    checkLoginStatus();
    loadPosts();
}); 