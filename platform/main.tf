terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Enable required APIs
resource "google_project_service" "pubsub" {
  service            = "pubsub.googleapis.com"
  disable_on_destroy = false
}

resource "google_project_service" "gmail" {
  service            = "gmail.googleapis.com"
  disable_on_destroy = false
}

# Note: Cloud Run and Secret Manager require billing to be enabled
# Uncomment these if you have billing enabled
# resource "google_project_service" "cloudrun" {
#   service            = "run.googleapis.com"
#   disable_on_destroy = false
# }

# resource "google_project_service" "secretmanager" {
#   service            = "secretmanager.googleapis.com"
#   disable_on_destroy = false
# }

# Create Pub/Sub topic for Gmail notifications
resource "google_pubsub_topic" "gmail_notifications" {
  name = var.pubsub_topic_name

  depends_on = [google_project_service.pubsub]
}

# Grant Gmail API permission to publish to the topic
resource "google_pubsub_topic_iam_member" "gmail_publisher" {
  topic  = google_pubsub_topic.gmail_notifications.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:gmail-api-push@system.gserviceaccount.com"
}

# Create Pub/Sub subscription (Push type)
resource "google_pubsub_subscription" "gmail_push" {
  name  = "${var.pubsub_topic_name}-subscription"
  topic = google_pubsub_topic.gmail_notifications.name

  push_config {
    push_endpoint = var.webhook_url

    # Optional: Add authentication if needed
    # oidc_token {
    #   service_account_email = google_service_account.pubsub_invoker.email
    # }
  }

  ack_deadline_seconds = 20

  retry_policy {
    minimum_backoff = "10s"
    maximum_backoff = "600s"
  }

  depends_on = [google_project_service.pubsub]
}

# Get project information
data "google_project" "project" {
  project_id = var.project_id
}

# Note: OAuth Client cannot be created automatically without an organization
# Please create it manually at: https://console.cloud.google.com/apis/credentials
# 1. Click "Create Credentials" > "OAuth 2.0 Client ID"
# 2. Application type: Web application
# 3. Add authorized redirect URI
# 4. Copy the Client ID and Secret to terraform.tfvars

# Generate credentials.json file
resource "local_file" "credentials_json" {
  filename = "${path.module}/../go/credentials.json"
  content = jsonencode({
    web = {
      client_id                   = var.oauth_client_id
      project_id                  = var.project_id
      auth_uri                    = "https://accounts.google.com/o/oauth2/auth"
      token_uri                   = "https://oauth2.googleapis.com/token"
      auth_provider_x509_cert_url = "https://www.googleapis.com/oauth2/v1/certs"
      client_secret               = var.oauth_client_secret
      redirect_uris = [var.oauth_redirect_uri]
    }
  })
  file_permission = "0600"
}

# Output important values
output "pubsub_topic_name" {
  description = "The full name of the Pub/Sub topic"
  value       = google_pubsub_topic.gmail_notifications.id
}

output "pubsub_subscription_name" {
  description = "The name of the Pub/Sub subscription"
  value       = google_pubsub_subscription.gmail_push.name
}

output "webhook_url" {
  description = "The webhook URL configured for push notifications"
  value       = var.webhook_url
}

output "credentials_file_path" {
  description = "Path to generated credentials.json"
  value       = local_file.credentials_json.filename
}

output "oauth_client_id" {
  description = "OAuth Client ID"
  value       = var.oauth_client_id
  sensitive   = true
}
