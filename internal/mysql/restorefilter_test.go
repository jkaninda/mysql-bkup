package mysql

import (
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func filter(in, target string, noDefiner bool) (string, error) {
	return filterFor(in, target, noDefiner, false)
}

func filterFor(in, target string, noDefiner, oracle bool) (string, error) {
	return filterWith(in, FilterOptions{Target: target, NoDefiner: noDefiner, OracleClient: oracle})
}

func filterWith(in string, o FilterOptions) (string, error) {
	rc := RestoreFilter(iotest.OneByteReader(strings.NewReader(in)), o)
	defer func() { _ = rc.Close() }()
	out, err := io.ReadAll(rc)
	return string(out), err
}

const sampleDump = "/*!40101 SET NAMES utf8 */;\n" +
	"-- Current Database: `classicmodels`\n" +
	"/*!40000 DROP DATABASE IF EXISTS `classicmodels`*/;\n" +
	"CREATE DATABASE /*!32312 IF NOT EXISTS*/`classicmodels` /*!40100 DEFAULT CHARACTER SET latin1 */;\n" +
	"\n" +
	"USE `classicmodels`;\n" +
	"CREATE TABLE `customers` (`id` int, `note` text);\n" +
	"INSERT INTO `customers` VALUES (1,'USE `other`;'),(2,'CREATE DEFINER=`a`@`b` VIEW');\n" +
	"/*!50001 CREATE ALGORITHM=UNDEFINED */\n" +
	"/*!50013 DEFINER=`u_83c8ac41`@`%` SQL SECURITY DEFINER */\n" +
	"/*!50001 VIEW `v` AS select 1 */;\n" +
	"DELIMITER ;;\n" +
	"CREATE DEFINER=`root`@`localhost` PROCEDURE `p`()\n" +
	"BEGIN\n" +
	"  SELECT 1;\n" +
	"END ;;\n" +
	"/*!50003 CREATE*/ /*!50017 DEFINER='app'@'%'*/ /*!50003 TRIGGER `t` BEFORE INSERT ON `customers` FOR EACH ROW SET NEW.id = 1 */;;\n" +
	"DELIMITER ;\n"

func TestRestoreFilterRetargets(t *testing.T) {
	out, err := filter(sampleDump, "mail", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, gone := range []string{"DROP DATABASE", "CREATE DATABASE", "\nUSE `classicmodels`;"} {
		if strings.Contains(out, gone) {
			t.Errorf("%q kept:\n%s", gone, out)
		}
	}
	if !strings.Contains(out, "'USE `other`;'") || !strings.Contains(out, "DEFINER=`root`") {
		t.Errorf("data or definers changed:\n%s", out)
	}
	if strings.Count(out, "\n") != strings.Count(sampleDump, "\n") {
		t.Errorf("line count changed")
	}

	// No target (all-in-one): untouched.
	if out, _ := filter(sampleDump, "", false); out != sampleDump {
		t.Errorf("all-in-one dump changed:\n%s", out)
	}
}

func TestRestoreFilterDefiners(t *testing.T) {
	out, err := filter(sampleDump, "mail", true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"/*!50013 SQL SECURITY DEFINER */\n",
		"CREATE PROCEDURE `p`()\n",
		"/*!50003 CREATE*/ /*!50017*/ /*!50003 TRIGGER",
		"(2,'CREATE DEFINER=`a`@`b` VIEW')", // data
	} {
		if !strings.Contains(out, want) {
			t.Errorf("want %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "u_83c8ac41") || strings.Contains(out, "`root`@") || strings.Contains(out, "'app'@") {
		t.Errorf("definer kept:\n%s", out)
	}
}

func TestRestoreFilterSeveralDatabases(t *testing.T) {
	_, err := filter("USE `a`;\nCREATE TABLE t(i int);\nUSE `b`;\n", "mail", false)
	if err == nil || !strings.Contains(err.Error(), "more than one database (a, b)") {
		t.Fatalf("err = %v", err)
	}
	if _, err := filter("CREATE DATABASE IF NOT EXISTS shop;\nuse shop\n", "mail", false); err != nil {
		t.Fatal(err)
	}
}

func TestCanSetDefiner(t *testing.T) {
	for grants, want := range map[string]bool{
		"GRANT USAGE ON *.* TO `u_1`@`%`\nGRANT ALL PRIVILEGES ON `blog`.* TO `u_1`@`%`":      false,
		"GRANT USAGE ON *.* TO `superman`@`%`":                                                false,
		"GRANT SELECT, INSERT, SUPER ON *.* TO `root`@`%` WITH GRANT OPTION":                  true,
		"GRANT USAGE ON *.* TO `a`@`%`\nGRANT SET_USER_ID,XA_RECOVER_ADMIN ON *.* TO `a`@`%`": true,
		"GRANT ALL PRIVILEGES ON *.* TO `root`@`localhost` WITH GRANT OPTION":                 true,
		"GRANT SET USER ON *.* TO `m`@`%`":                                                    true,
	} {
		if got := canSetDefiner(grants); got != want {
			t.Errorf("%q: %v", grants, got)
		}
	}
}

func TestRestoreFilterSandbox(t *testing.T) {
	in := "/*M!999999\\- enable the sandbox mode */ \n-- MariaDB dump 10.20\nCREATE TABLE t (i int);\n"
	if out, _ := filterFor(in, "mail", false, true); out != "\n-- MariaDB dump 10.20\nCREATE TABLE t (i int);\n" {
		t.Fatalf("oracle client: %q", out)
	}
	if out, _ := filterFor(in, "mail", false, false); out != in {
		t.Fatalf("mariadb client: %q", out)
	}
}

func TestRestoreFilterCollations(t *testing.T) {
	in := "  `name` varchar(100) COLLATE utf8mb4_uca1400_ai_ci DEFAULT NULL,\n" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_as_cs;\n" +
		"/*!50003 SET collation_connection  = utf8mb4_uca1400_ai_ci */ ;\n" +
		"INSERT INTO `t` VALUES ('utf8mb4_uca1400_ai_ci');\n" +
		"  `x` text COLLATE utf8mb4_uca1400_swedish_ai_ci,\n"
	want := "  `name` varchar(100) COLLATE utf8mb4_0900_ai_ci DEFAULT NULL,\n" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_as_cs;\n" +
		"/*!50003 SET collation_connection  = utf8mb4_0900_ai_ci */ ;\n" +
		"INSERT INTO `t` VALUES ('utf8mb4_uca1400_ai_ci');\n" +
		"  `x` text COLLATE utf8mb4_uca1400_swedish_ai_ci,\n"
	if out, _ := filterWith(in, FilterOptions{Target: "mail", MySQLServer: true}); out != want {
		t.Fatalf("got:\n%s", out)
	}
	if out, _ := filterWith(in, FilterOptions{Target: "mail"}); out != in {
		t.Fatalf("MariaDB target changed:\n%s", out)
	}
}

func TestRestoreFilterSQLModes(t *testing.T) {
	in := "/*!50003 SET sql_mode              = 'STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION' */ ;\n" +
		"/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO,SIMULTANEOUS_ASSIGNMENT' */;\n" +
		"/*!50003 SET sql_mode              = @saved_sql_mode */ ;\n" +
		"INSERT INTO `t` VALUES ('sql_mode=''NO_AUTO_CREATE_USER''');\n"
	want := "/*!50003 SET sql_mode              = 'STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION' */ ;\n" +
		"/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;\n" +
		"/*!50003 SET sql_mode              = @saved_sql_mode */ ;\n" +
		"INSERT INTO `t` VALUES ('sql_mode=''NO_AUTO_CREATE_USER''');\n"
	if out, _ := filterWith(in, FilterOptions{Target: "mail", MySQLServer: true}); out != want {
		t.Fatalf("got:\n%s", out)
	}
	if out, _ := filterWith(in, FilterOptions{Target: "mail"}); out != in {
		t.Fatalf("MariaDB target changed:\n%s", out)
	}
}
