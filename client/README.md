# bidking Unity 客户端

目标 Unity 版本：`6000.4.7f1`。

安装目标 Unity 版本后，通过 Unity Hub 打开这个目录：

```text
D:\qt\projects\game\client
```

如果 Unity Hub 没有反应，可以先移除 Hub 里已有的旧项目记录，再重新选择这个 `client` 目录添加。也可以直接用已安装的 Unity Editor 打开：

```powershell
& "D:\unity\Editor\6000.4.7f1\Editor\Unity.exe" -projectPath "D:\qt\projects\game\client"
```

推荐工作流：

- 用 Unity Editor 管理场景、UI、预制体、资源和打包。
- 用 VSCode 编写 C# 脚本。
- 游戏逻辑统一放在 `Assets/_Project/Scripts`。
- `Library` 等 Unity 生成目录不要提交到 Git。
