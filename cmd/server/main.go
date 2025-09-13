package server

import (
	"ckks-credit/internal/app"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"os"
)

func main() {
	// 환경 변수 설정 (기본값 포함)
	dbHost := env("DB_HOST", "localhost")
	dbPort := env("DB_PORT", "3306")
	dbUser := env("DB_USER", "root")
	dbPassword := env("DB_PASSWORD", "password")
	dbName := env("DB_NAME", "creditdb")

	// DSN 구성
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// MySQL 연결
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("DB 연결 실패: %v", err)
	}
	defer db.Close()

	// 연결 테스트
	if err := db.Ping(); err != nil {
		log.Fatalf("DB Ping 실패: %v", err)
	}
	log.Println("DB 연결 성공")

	// 애플리케이션 초기화
	application, err := app.NewApplication()
	if err != nil {
		log.Fatalf("애플리케이션 초기화 실패: %v", err)
	}

	// 애플리케이션 시작
	if err := application.Start(); err != nil {
		log.Fatalf("애플리케이션 시작 실패: %v", err)
	}

	// REST API 핸들러 등록
	http.HandleFunc("/api/v1/health", healthCheckHandler)
	http.HandleFunc("/api/v1/score/encrypt", encryptHandler(application))
	http.HandleFunc("/api/v1/score/infer", inferenceHandler(application))

	// 서버 시작
	port := env("PORT", "8080")
	log.Printf("서버 시작 중... 포트: %s", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("서버 실행 실패: %v", err)
	}
}

// 핸들러 함수들
func createUser(c *gin.Context) {
	// 사용자 생성 로직
	c.JSON(http.StatusOK, gin.H{"message": "사용자가 생성되었습니다"})
}

func getUser(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id, "message": "사용자 정보 조회 완료"})
}

func encryptFeatures(c *gin.Context) {
	// 특성 암호화 로직
	c.JSON(http.StatusOK, gin.H{"message": "특성이 암호화되었습니다"})
}

func getEncryptedFeatures(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id, "message": "암호화된 특성 조회 완료"})
}

func evaluateCredit(c *gin.Context) {
	// 신용 평가 로직
	c.JSON(http.StatusOK, gin.H{"message": "신용 평가가 완료되었습니다"})
}

func getCreditResult(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id, "message": "신용 평가 결과 조회 완료"})
}
