# Examples

本目录包含各种使用场景的示例 hook 脚本。

## 目录

### 1. [ci-cd-trigger.sh](./ci-cd-trigger.sh)

触发 CI/CD 流水线的示例脚本。

**功能：**
- 检测新上传的 artifact 文件
- 自动触发 Jenkins/GitLab CI 流水线
- 支持环境区分（生产/测试）

**使用场景：**
- 自动化部署流程
- 持续集成触发
- Artifact 上传后自动测试

### 2. [alist-sync.sh](./alist-sync.sh)

AList 索引同步示例脚本。

**功能：**
- 文件变更时自动刷新 AList 索引
- 支持目录级别刷新
- API 错误处理

**使用场景：**
- 文件管理系统集成
- 网盘索引自动更新
- NAS 文件共享平台

### 3. [notifications.sh](./notifications.sh)

多渠道通知示例脚本。

**功能：**
- Slack 通知
- 邮件通知
- 事件类型过滤
- 自定义消息格式

**使用场景：**
- 文件上传提醒
- 重要变更通知
- 团队协作

### 4. [backup-files.sh](./backup-files.sh)

自动备份示例脚本。

**功能：**
- 文件上传时自动创建备份
- 版本化备份（带时间戳）
- 自动清理旧备份
- 备份大小统计

**使用场景：**
- 数据保护
- 版本控制
- 灾难恢复

## 快速开始

### 1. 选择示例脚本

```bash
# 复制示例脚本
cp examples/ci-cd-trigger.sh /usr/local/bin/my-hook.sh

# 编辑配置
vim /usr/local/bin/my-hook.sh

# 设置权限
chmod +x /usr/local/bin/my-hook.sh
```

### 2. 配置 inotify-hook

```bash
# 测试运行
inotify-hook watch /data/artifacts \
    --mode=write-complete \
    --hook=/usr/local/bin/my-hook.sh \
    --verbose

# 生产部署
inotify-hook watch /data/artifacts \
    --mode=write-complete \
    --hook=/usr/local/bin/my-hook.sh \
    --debounce=1000
```

### 3. 配置 systemd 服务

```bash
# 编辑服务文件
vim /etc/systemd/system/inotify-hook.service

# 更新 ExecStart 中的 hook 路径
ExecStart=/usr/local/bin/inotify-hook watch \
    /data/artifacts \
    --mode=write-complete \
    --hook=/usr/local/bin/my-hook.sh \
    --debounce=1000

# 重载并重启服务
systemctl daemon-reload
systemctl restart inotify-hook
```

## 自定义 Hook 脚本

### Hook 参数

Hook 脚本接收以下参数：

```bash
$1 - EVENT_TYPE    # 事件类型 (CLOSE_WRITE, MOVED_TO, CREATE, DELETE, etc.)
$2 - FILE_PATH     # 文件完整路径
$3 - FILE_NAME     # 文件名
$4 - IS_DIR        # 是否为目录 (true/false)
```

### 基础模板

```bash
#!/bin/bash
set -euo pipefail

EVENT_TYPE="$1"
FILE_PATH="$2"
FILE_NAME="$3"
IS_DIR="$4"

# 你的逻辑代码
echo "Event: $EVENT_TYPE, File: $FILE_PATH"
```

### 最佳实践

1. **错误处理**
   ```bash
   set -euo pipefail  # 遇到错误立即退出
   ```

2. **日志记录**
   ```bash
   log() {
       echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" >> /var/log/hook.log
   }
   ```

3. **事件过滤**
   ```bash
   # 仅处理文件事件
   if [[ "$IS_DIR" == "true" ]]; then
       exit 0
   fi

   # 仅处理特定文件类型
   if [[ "$FILE_PATH" != *.tar.gz ]]; then
       exit 0
   fi
   ```

4. **并发控制**
   ```bash
   # 使用文件锁防止并发执行
   LOCK_FILE="/tmp/hook.lock"
   (
       flock -n 9 || exit 1
       # 你的代码
   ) 9>"$LOCK_FILE"
   ```

5. **超时控制**
   ```bash
   # 设置超时时间（秒）
   timeout 300 your_command
   ```

## 调试技巧

### 1. 详细日志

```bash
inotify-hook watch /data \
    --hook=/usr/local/bin/my-hook.sh \
    --verbose  # 显示详细事件日志
```

### 2. 手动测试 Hook

```bash
# 模拟 CLOSE_WRITE 事件
/usr/local/bin/my-hook.sh CLOSE_WRITE /data/test.txt test.txt false
```

### 3. 查看服务日志

```bash
# 查看实时日志
journalctl -u inotify-hook -f

# 查看最近 100 行
journalctl -u inotify-hook -n 100
```

## 贡献示例

欢迎贡献更多示例脚本！请遵循以下规范：

1. 添加详细注释
2. 包含使用场景说明
3. 提供配置示例
4. 错误处理完善
5. 遵循 Shell 脚本最佳实践

提交 PR 时请更新本 README。
