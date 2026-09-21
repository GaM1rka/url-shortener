package postgres

const (
	getByShortQuery = `
		SELECT short_code, original_url
		FROM links
		WHERE short_code = $1
	`

	getByOriginalQuery = `
		SELECT short_code, original_url
		FROM links
		WHERE original_url = $1
	`

	createLinkQuery = `
		INSERT INTO links (short_code, original_url)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`

	getConflictQuery = `
		SELECT
			EXISTS(SELECT 1 FROM links WHERE original_url = $1),
			EXISTS(SELECT 1 FROM links WHERE short_code = $2)
	`
)
