document.addEventListener('DOMContentLoaded', function() {
    const postDetail = document.getElementById('postDetail');
    const pageTitle = document.getElementById('pageTitle');
    
    const pathname = window.location.pathname.replace(/\/+$/, "");
    const segs = pathname.split('/').filter(Boolean);
    const postId = segs.length >= 2 ? segs[segs.length - 1] : null;
    
    if (!postId) {
        postDetail.innerHTML = '<p>文章 ID 无效</p>';
        return;
    }

    // 检查登录状态
    let isLoggedIn = false;
    checkLoginStatus().then(v => isLoggedIn = v).catch(() => {});
    
    loadPostDetail(postId);
    
    async function loadPostDetail(id) {
        try {
            const response = await fetch(`/api/post?id=${encodeURIComponent(id)}`, {
                headers: { 'Accept': 'application/json' }
            });
            if (!response.ok) {
                const errText = await response.text().catch(() => '');
                throw new Error(`HTTP ${response.status}: ${errText}`);
            }
            const post = await response.json();
            // 更新页面标题
            pageTitle.textContent = `${post.title} - Fenger Journal`;

            const adminBar = isLoggedIn ? `
                <div class="admin-controls" style="justify-content:flex-start;margin:12px 0 0 0;gap:8px;">
                    <button class="btn btn-secondary" id="editBtn">编辑</button>
                    <button class="btn btn-secondary" id="deleteBtn">删除</button>
                </div>` : '';

            postDetail.innerHTML = `
                <header class="post-header">
                    <h1 class="post-title">${post.title}</h1>
                    <div class="post-meta">${formatDate(post.created_at)}</div>
                    ${adminBar}
                </header>
                <div class="post-content" id="postContent">${post.html_content}</div>
            `;

            if (isLoggedIn) {
                wireAdminActions(post);
            }
        } catch (error) {
            console.error('Load post error:', error);
            postDetail.innerHTML = `
                <div class="error-state">
                    <p>文章加载失败</p>
                    <pre style="white-space:pre-wrap;color:#666;font-size:12px;">${String(error)}</pre>
                    <a href="/" class="btn btn-primary">返回首页</a>
                </div>
            `;
        }
    }

    function wireAdminActions(post) {
        const editBtn = document.getElementById('editBtn');
        const deleteBtn = document.getElementById('deleteBtn');
        const contentEl = document.getElementById('postContent');

        let editing = false;
        let editor;

        editBtn.addEventListener('click', async () => {
            if (!editing) {
                editing = true;
                // 简单编辑器：标题输入 + Markdown 文本域
                const currentMd = await fetchRawContent(post.id).catch(() => '');
                contentEl.innerHTML = `
                    <div class="form-group">
                        <label>标题</label>
                        <input type="text" id="editTitle" value="${escapeHtml(post.title)}" class="input" style="width:100%;padding:10px;border:1px solid #e5e5e5;border-radius:6px;">
                    </div>
                    <div class="form-group" style="margin-top:12px;">
                        <label>内容 (Markdown)</label>
                        <textarea id="editContent" rows="12" style="width:100%;padding:10px;border:1px solid #e5e5e5;border-radius:6px;">${escapeHtml(currentMd || '')}</textarea>
                    </div>
                    <div class="form-actions" style="margin-top:12px;gap:8px;">
                        <button class="btn btn-primary" id="saveBtn">保存</button>
                        <button class="btn btn-secondary" id="cancelBtn">取消</button>
                    </div>
                `;
                bindSaveCancel(post.id);
            }
        });

        deleteBtn.addEventListener('click', async () => {
            if (!confirm('确定删除该文章吗？')) return;
            try {
                const resp = await fetch(`/post/${encodeURIComponent(post.id)}`, { method: 'DELETE' });
                if (!resp.ok && resp.status !== 204) {
                    const t = await resp.text().catch(()=> '');
                    throw new Error(`HTTP ${resp.status}: ${t}`);
                }
                window.location.href = '/';
            } catch (e) {
                alert('删除失败');
                console.error(e);
            }
        });
    }

    function bindSaveCancel(id) {
        const saveBtn = document.getElementById('saveBtn');
        const cancelBtn = document.getElementById('cancelBtn');
        const contentEl = document.getElementById('postContent');

        saveBtn.addEventListener('click', async () => {
            const title = document.getElementById('editTitle').value.trim();
            const content = document.getElementById('editContent').value;
            if (!title) { alert('标题不能为空'); return; }
            try {
                const resp = await fetch(`/post/${encodeURIComponent(id)}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ title, content })
                });
                if (!resp.ok) {
                    const t = await resp.text().catch(()=> '');
                    throw new Error(`HTTP ${resp.status}: ${t}`);
                }
                const updated = await resp.json();
                pageTitle.textContent = `${updated.title} - Fenger Journal`;
                contentEl.innerHTML = updated.html_content;
                // 刷新以展示最新渲染
                window.location.reload();
            } catch (e) {
                alert('保存失败');
                console.error(e);
            }
        });

        cancelBtn.addEventListener('click', () => {
            window.location.reload();
        });
    }

    async function fetchRawContent(id) {
        // 这里简单复用已返回的 HTML 不含原 Markdown。若需要原始内容，可新增 /api/post/raw 接口。
        // 暂时返回空，让编辑器用空白，或后续可从 /posts 列表缓存中取。
        return '';
    }

    async function checkLoginStatus() {
        try {
            const resp = await fetch('/status');
            const data = await resp.json();
            return !!data.logged_in;
        } catch (_) { return false; }
    }

    function escapeHtml(s) {
        return (s || '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;','\'':'&#39;'}[c]));
    }

    function formatDate(dateString) {
        const date = new Date(dateString);
        const now = new Date();
        const diffTime = Math.abs(now - date);
        const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
        if (diffDays === 1) return '昨天';
        if (diffDays < 7) return `${diffDays} 天前`;
        if (diffDays < 30) return `${Math.floor(diffDays / 7)} 周前`;
        if (diffDays < 365) return `${Math.floor(diffDays / 30)} 个月前`;
        return date.toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' });
    }
}); 