# File Upload

The storage interface supports local development storage and S3-compatible deployments, including Cloudflare R2 and MinIO configurations.

Production uploads should validate MIME types, enforce size limits, prevent path traversal, use private buckets for sensitive files, and issue signed URLs for restricted access.
