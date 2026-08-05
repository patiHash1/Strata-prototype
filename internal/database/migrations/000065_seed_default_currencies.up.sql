INSERT INTO currencies (code, name, symbol, decimal_places) VALUES
					('USD', 'US Dollar', '$', 2),
					('EUR', 'Euro', '€', 2),
					('GBP', 'British Pound', '£', 2),
					('JPY', 'Japanese Yen', '¥', 0),
					('INR', 'Indian Rupee', '₹', 2),
					('CAD', 'Canadian Dollar', 'C$', 2),
					('AUD', 'Australian Dollar', 'A$', 2),
					('CHF', 'Swiss Franc', 'Fr', 2),
					('CNY', 'Chinese Yuan', '¥', 2),
					('BRL', 'Brazilian Real', 'R$', 2)
				ON CONFLICT (code) DO NOTHING
