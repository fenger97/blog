document.addEventListener('DOMContentLoaded', function() {
    const postDetail = document.getElementById('postDetail');
    const pageTitle = document.getElementById('pageTitle');
    
    // 从 URL 中获取文章 ID
    const pathParts = window.location.pathname.split('/');
    const postId = pathParts[pathParts.length - 1];
    
    if (!postId) {
        postDetail.innerHTML = '<p>文章 ID 无效</p>';
        return;
    }
    
    // 加载文章详情
    loadPostDetail(postId);
    
    async function loadPostDetail(id) {
        try {
            const response = await fetch(`/post/${id}`, {
                headers: {
                    'Accept': 'application/json'
                }
            });
            if (!response.ok) {
                throw new Error('文章不存在');
            }
            
            const post = await response.json();
            
            // 更新页面标题
            pageTitle.textContent = `${post.title} - 我的博客`;
            
            // 显示文章内容
            postDetail.innerHTML = `
                <header class="post-header">
                    <h1 class="post-title">${post.title}</h1>
                    <div class="post-meta">
                        ${formatDate(post.created_at)}
                    </div>
                </header>
                <div class="post-content">
                    ${post.html_content}
                </div>
            `;
        } catch (error) {
            console.error('Error:', error);
            postDetail.innerHTML = `
                <div class="error-state">
                    <p>文章加载失败</p>
                    <a href="/" class="btn btn-primary">返回首页</a>
                </div>
            `;
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
}); 