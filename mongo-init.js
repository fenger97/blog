// 创建数据库和用户
db = db.getSiblingDB('blog');

// 创建用户（可选，用于生产环境）
db.createUser({
  user: 'blog_user',
  pwd: 'blog_password',
  roles: [
    {
      role: 'readWrite',
      db: 'blog'
    }
  ]
});

// 创建 posts 集合的索引
db.posts.createIndex({ "created_at": -1 });
db.posts.createIndex({ "title": "text", "content": "text" });

print('Blog database initialized successfully!'); 