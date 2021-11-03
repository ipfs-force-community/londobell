curl https://oapi.dingtalk.com/robot/send?access_token=$DINGTALK_BOT_ACCESS_TOKEN \
-H 'Content-Type: application/json' \
-d "{\"msgtype\": \"markdown\", \"markdown\": {\"title\": \"Github Action Failure Alarm\", \"text\":\"### 日三省吾身\\n[PR $GITHUB_PR_NUM](https://github.com/ipfs-force-community/londobell/pull/$GITHUB_PR_NUM)\"}}"
