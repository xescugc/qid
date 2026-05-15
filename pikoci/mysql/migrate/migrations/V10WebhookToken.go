package migrations

var V10WebhookToken = Migration{
	Name: "WebhookToken",
	SQL:  `ALTER TABLE resources ADD COLUMN webhook_token VARCHAR(36) DEFAULT '';`,
}
