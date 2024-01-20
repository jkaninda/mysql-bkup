package utils

const Notice = "Please remove --operation flag.\n" +
	"Use: \n" +
	"- backup for database backup operation [eg: bkup backup -d  database_name ...]\n" +
	"- restore for database restore operation [eg. bkup restore -d database_name ...]\n" +
	"We are sorry for this inconvenient"
const RestoreExample = "mysql-bkup restore --dbname database --file db_20231219_022941.sql.gz\n" +
	"bkup restore --dbname database --storage s3 --path /custom-path --file db_20231219_022941.sql.gz"
const BackupExample = "mysql-bkup backup --dbname database --disable-compression\n" +
	"mysql-bkup backup --dbname database --storage s3 --path /custom-path --disable-compression"

const MainExample = "mysql-bkup backup --dbname database --disable-compression\n" +
	"mysql-bkup backup --dbname database --storage s3 --path /custom-path\n" +
	"mysql-bkup restore --dbname database --file db_20231219_022941.sql.gz"
