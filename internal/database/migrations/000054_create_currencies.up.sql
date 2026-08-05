CREATE TABLE IF NOT EXISTS currencies (
					code VARCHAR(3) PRIMARY KEY,
					name VARCHAR(100) NOT NULL,
					symbol VARCHAR(10) NOT NULL,
					decimal_places INT DEFAULT 2,
					is_active BOOLEAN DEFAULT TRUE
				)
