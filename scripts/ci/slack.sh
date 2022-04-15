set -eo pipefail

# Populate header
slack_msg_header=":white_check_mark: Deploy to ${CI_ENVIRONMENT_TIER}/${CI_ENVIRONMENT_NAME} succeeded"
if [[ -n "${FAILED}" ]]; then
  slack_msg_header=":x: Deploy to ${CI_ENVIRONMENT_TIER}/${CI_ENVIRONMENT_NAME} failed"
fi

# Populate slack message body
slack_msg_body=$(cat <<-END
app: ${CI_PROJECT_TITLE}
url: ${INFO_URL}
deploy_user: ${GITLAB_USER_LOGIN}
tag: ${CI_COMMIT_TAG}
job: ${CI_JOB_URL}
END
)

slack_summary=$(cat <<-SLACK
{
  "icon_url": "https://about.gitlab.com/images/press/logo/png/gitlab-icon-rgb.png",
  "username": "GitLab",
  "blocks": [
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "${slack_msg_header}"
      }
    },
    {
      "type": "divider"
    },
    {
      "type": "section",
      "text": {
        "type": "mrkdwn",
        "text": "${slack_msg_body}"
      }
    }
  ]
}
SLACK
)

curl -X POST --data-urlencode "payload=${slack_summary}" https://hooks.slack.com/services/${SLACK_TOKEN}
