// Package utils /
/*****
@author    Jonas Kaninda
@license   MIT License <https://opensource.org/licenses/MIT>
@Copyright © 2024 Jonas Kaninda
**/
package utils

const RestoreExample = "mysql-bkup restore --dbname database --file db_20231219_022941.sql.gz\n" +
	"restore --dbname database --storage s3 --path /custom-path --file db_20231219_022941.sql.gz"
const BackupExample = "mysql-bkup backup --dbname database --disable-compression\n" +
	"backup --dbname database --storage s3 --path /custom-path --disable-compression"

const MainExample = "mysql-bkup backup --dbname database --disable-compression\n" +
	"backup --dbname database --storage s3 --path /custom-path\n" +
	"restore --dbname database --file db_20231219_022941.sql.gz"
