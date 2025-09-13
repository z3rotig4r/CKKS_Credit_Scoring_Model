package app

import (
	"fmt"
	"log"
)

type Application struct {
	CreditService *CreditScoringService
}

// NewApplication 애플리케이션 초기화
func NewApplication() (*Application, error) {
	log.Println("Application initialized...")

	// 신용점수 서비스 초기화
	creditService, err := NewCreditScoringService()
	if err != nil {
		return nil, fmt.Errorf("Credit Score Service Initialization Failed!: %v", err)
	}

	app := &Application{
		CreditService: creditService,
	}

	log.Println("Application initialization succeeded")
	return app, nil
}

// Start 애플리케이션 시작
func (app *Application) Start() error {
	log.Println("Application starting...")

	// 여기서 추가 초기화 작업 수행
	// 예: DB 연결 확인, 키 로딩 등

	return nil
}

// Stop 애플리케이션 종료
func (app *Application) Stop() error {
	log.Println("애플리케이션 서비스 종료")

	// 정리 작업 수행
	// 예: 연결 해제, 리소스 정리 등

	return nil
}
