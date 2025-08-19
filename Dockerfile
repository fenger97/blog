# 运行阶段
FROM alpine:latest

# 创建非 root 用户
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY ./blog-app /app/blog-app 

# 复制静态文件
COPY ./static /app/static 

# 切换到非 root 用户
USER appuser

# 暴露端口
EXPOSE 1834

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:1834/ || exit 1

# 启动应用
CMD ["./blog-app"] 
