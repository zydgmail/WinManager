@echo off
echo ���ڿ��� 50051 �� 50052 �� TCP �� UDP �˿�...

powershell -Command "New-NetFirewallRule -DisplayName 'Open Port 50051 TCP' -Direction Inbound -LocalPort 50051 -Protocol TCP -Action Allow"
powershell -Command "New-NetFirewallRule -DisplayName 'Open Port 50051 UDP' -Direction Inbound -LocalPort 50051 -Protocol UDP -Action Allow"
powershell -Command "New-NetFirewallRule -DisplayName 'Open Port 50052 TCP' -Direction Inbound -LocalPort 50052 -Protocol TCP -Action Allow"
powershell -Command "New-NetFirewallRule -DisplayName 'Open Port 50052 UDP' -Direction Inbound -LocalPort 50052 -Protocol UDP -Action Allow"

echo ��ɣ�
pause
