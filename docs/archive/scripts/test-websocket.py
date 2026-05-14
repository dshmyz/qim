#!/usr/bin/env python3
# test-websocket.py
# WebSocket 功能自动化测试
# 用法: QIM_TOKEN=<token> python3 test-websocket.py

import asyncio
import json
import os
import sys
import websockets

BASE_URL = os.environ.get("QIM_BASE_URL", "ws://localhost:8080")
TOKEN = os.environ.get("QIM_TOKEN", "")

if not TOKEN:
    print("❌ 请设置 QIM_TOKEN 环境变量")
    print("   示例: QIM_TOKEN=<your_token> python3 test-websocket.py")
    sys.exit(1)

PASS = 0
FAIL = 0

def assert_result(condition, test_name):
    global PASS, FAIL
    if condition:
        print(f"  ✅ PASS: {test_name}")
        PASS += 1
    else:
        print(f"  ❌ FAIL: {test_name}")
        FAIL += 1

async def test_websocket():
    global PASS, FAIL
    print("===== WebSocket 功能测试 =====")
    
    uri = f"{BASE_URL}/api/v1/ws?token={TOKEN}"
    
    try:
        async with websockets.connect(uri) as ws:
            print("  ✅ PASS: WebSocket 连接成功")
            PASS += 1
            
            # WS-002: 发送心跳
            await ws.send(json.dumps({"type": "heartbeat", "data": {}}))
            # 心跳不需要服务端响应验证
            
            # WS-003: 发送消息
            msg = json.dumps({
                "type": "send_message",
                "data": {
                    "conversation_id": 1,
                    "type": "text",
                    "content": "WebSocket测试消息"
                },
                "request_id": "test-001"
            })
            await ws.send(msg)
            
            # 等待 ACK
            try:
                resp = await asyncio.wait_for(ws.recv(), timeout=5)
                data = json.loads(resp)
                assert_result(
                    data.get("type") in ["ack", "new_message"],
                    "发送消息收到响应"
                )
            except asyncio.TimeoutError:
                assert_result(False, "发送消息超时")
            
    except websockets.exceptions.ConnectionClosed:
        print("  ❌ FAIL: WebSocket 连接被关闭")
        FAIL += 1
    except Exception as e:
        print(f"  ❌ FAIL: WebSocket 异常: {e}")
        FAIL += 1
    
    print(f"\n===== WebSocket 测试结果: {PASS} 通过, {FAIL} 失败 =====")

asyncio.run(test_websocket())
