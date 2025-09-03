@echo off
echo 正在开启 50051 和 50052 的 TCP 和 UDP 端口...

powershell -Command "New-NetFirewallRule -DisplayName 'Open Port 50051 TCP' -Direction Inbound -LocalPort 50051 -Protocol TCP -Action Allow"
powershell -Command "New-NetFirewallRule -DisplayName 'Open Port 50051 UDP' -Direction Inbound -LocalPort 50051 -Protocol UDP -Action Allow"
powershell -Command "New-NetFirewallRule -DisplayName 'Open Port 50052 TCP' -Direction Inbound -LocalPort 50052 -Protocol TCP -Action Allow"
powershell -Command "New-NetFirewallRule -DisplayName 'Open Port 50052 UDP' -Direction Inbound -LocalPort 50052 -Protocol UDP -Action Allow"

echo 完成！
pause
