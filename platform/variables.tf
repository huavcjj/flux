variable "project_id" {
  description = "Google Cloud Project ID"
  type        = string
}

variable "region" {
  description = "Google Cloud region"
  type        = string
  default     = "asia-northeast1"
}

variable "pubsub_topic_name" {
  description = "Name of the Pub/Sub topic for Gmail notifications"
  type        = string
  default     = "gmail-notifications"
}

variable "webhook_url" {
  description = "Webhook URL for push notifications (e.g., https://your-domain.com/webhook/pubsub)"
  type        = string
}

variable "line_channel_token" {
  description = "LINE Messaging API Channel Token"
  type        = string
  sensitive   = true
}

variable "line_channel_secret" {
  description = "LINE Messaging API Channel Secret"
  type        = string
  sensitive   = true
}

variable "oauth_redirect_uri" {
  description = "OAuth 2.0 redirect URI for Gmail authentication"
  type        = string
}

variable "oauth_client_id" {
  description = "OAuth 2.0 Client ID (create manually at console.cloud.google.com/apis/credentials)"
  type        = string
}

variable "oauth_client_secret" {
  description = "OAuth 2.0 Client Secret"
  type        = string
  sensitive   = true
}
