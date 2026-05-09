# This script is for CI notification (internal use only)
# It sends notifications to DingTalk when PR actions fail
# To use: set DINGTALK_BOT_ACCESS_TOKEN environment variable

curl https://oapi.dingtalk.com/robot/send?access_token=$DINGTALK_BOT_ACCESS_TOKEN \
-H 'Content-Type: application/json' \
-d "{\"msgtype\": \"markdown\", \"markdown\": {\"title\": \"Github Action Failure Alarm\", \"text\":\"### PR Failed\\n[PR \$GITHUB_PR_NUM](https://github.com/ipfs-force-community/londobell/pull/\$GITHUB_PR_NUM)\"}}}"
