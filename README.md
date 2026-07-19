开发记录
1. 本次目标
本次目标是验证并实现一个 Windows 隐私屏：
- 被控端的 PVE 画面显示黑屏。
- 控制端和录屏程序仍然可以看到原始桌面。
- GStreamer 使用 d3d11screencapturesrc 捕获桌面。
- 捕获结果保存为 MP4 文件。

2. 环境
当前测试环境是 Windows 虚拟机，通过 PVE 访问。
PVE 在测试中可以作为“类似物理显示器”的观察端：
- 开启隐私屏后，PVE 能看到黑屏。
- GStreamer 在 PVE/ 桌面会话中可以录到原始桌面。
但 PVE 和真实物理显示器仍然可能存在差异。

3. ToDesk 行为验证
测试过 ToDesk 隐私屏场景：
- 开启 ToDesk 隐私屏后，控制端仍能看到画面。
- 在被控端执行屏幕捕获时，可以捕获到符合预期的桌面画面。
这个结果说明，在当前 PVE 环境下，ToDesk 隐私屏的整体行为和需求描述基本一致。

4. 隐私屏实现方式
当前 Go 程序的实现方式是软件遮盖：
- 创建一个全屏置顶窗口，覆盖整个虚拟屏幕区域。
- 使用 GDI+ 加载 privacy-screen.png，并绘制到这个遮盖窗口上。
- 调用 Windows API：SetWindowDisplayAffinity
- 设置：WDA_EXCLUDEFROMCAPTURE

SetWindowDisplayAffinity 的作用是让 Windows 支持的屏幕捕获接口不要捕获这个遮盖窗口。

所以当前效果是：
PVE看到：指定图片
GStreamer 录到：原始桌面

使用时，把要显示的图片放到项目目录，并命名为：
privacy-screen.png

然后双击 start-privacy-screen.bat 即可。
如果没有 privacy-screen.png，程序会自动退回黑屏。

5. 当前项目文件
主要文件：
privacy_screen.go                  隐私屏 Go 源码
build.bat                          双击编译
start-privacy-screen.bat           双击开启隐私屏
stop-privacy-screen.bat            双击关闭隐私屏
build.ps1                          build.bat 调用的编译脚本

6. 当前使用流程
6.1 编译
双击 build.bat 生成 privacy-screen.exe 

6.2 开启隐私屏
如果要显示图片，先把图片放到项目目录，并命名为：
privacy-screen.png

在 PVE 桌面里双击：
start-privacy-screen.bat
（也可以在这里定时关闭）

如果没有 privacy-screen.png，会显示黑屏。

6.3 通过 SSH 关闭隐私屏
隐私屏开启后，PVE 桌面已经变成黑屏，无法方便地在桌面里双击 stop-privacy-screen.bat，可以通过 SSH 关闭。
SSH 登录 Windows 后，进入项目目录：
cd C:\Users\admin\Desktop\privacy-screen-capture

然后运行：
powershell -ExecutionPolicy Bypass -File .\scripts\ssh-stop-privacy-screen.ps1

说明：
不要在 SSH 里直接运行 privacy-screen.exe off，因为 SSH 和 PVE 桌面属于不同会话，SSH 里可能找不到 PVE 桌面中的黑屏窗口。

7. 当前结论
本次测试已经验证：
- Go 程序可以开启软件隐私屏。
- PVE 可以看到黑屏。
- GStreamer 使用 d3d11screencapturesrc 可以在隐私屏开启期间录到原始桌面。
