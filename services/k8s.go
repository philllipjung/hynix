package services

import (
	"context"
	"fmt"
	"log"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var k8sClient client.Client

// initK8sClient - Kubernetes 클라이언트 초기화
func initK8sClient() error {
	if k8sClient != nil {
		return nil
	}

	// 클러스터 내부에서 실행 시 in-cluster config 사용
	cfg, err := rest.InClusterConfig()
	if err != nil {
		// 로컬 개발 환경을 위해 kubeconfig 사용 시도
		cfg, err = config.GetConfig()
		if err != nil {
			return fmt.Errorf("Kubernetes config 로드 실패: %w", err)
		}
		// TLS 인증서 검증 건너뛰기 (minikube 개발 환경)
		cfg.TLSClientConfig.Insecure = true
		cfg.TLSClientConfig.CAFile = ""
	}

	// 클라이언트 생성
	k8sClient, err = client.New(cfg, client.Options{})
	if err != nil {
		return fmt.Errorf("Kubernetes 클라이언트 생성 실패: %w", err)
	}

	log.Println("Kubernetes 클라이언트 초기화 완료")
	return nil
}

// CreateSparkApplicationCRFromYAML - YAML 문자열로 Kubernetes에 SparkApplication CR 생성
func CreateSparkApplicationCRFromYAML(yamlStr string) (*CreateResult, error) {
	// 클라이언트 초기화
	if err := initK8sClient(); err != nil {
		return nil, err
	}

	ctx := context.Background()

	// YAML을 Unstructured로 파싱
	u := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(yamlStr), u); err != nil {
		return nil, fmt.Errorf("YAML 파싱 실패: %w", err)
	}

	// GVK 설정
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "sparkoperator.k8s.io",
		Version: "v1beta2",
		Kind:    "SparkApplication",
	})

	// 이름과 네임스페이스 추출
	name := u.GetName()
	namespace := u.GetNamespace()

	if name == "" {
		return nil, fmt.Errorf("이름이 없습니다")
	}
	if namespace == "" {
		namespace = "default"
		u.SetNamespace(namespace)
	}

	// 이미 존재하는지 확인
	var existing unstructured.Unstructured
	existing.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "sparkoperator.k8s.io",
		Version: "v1beta2",
		Kind:    "SparkApplication",
	})

	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}, &existing)

	if err == nil {
		// 이미 존재하면 삭제 후 재생성
		if err := k8sClient.Delete(ctx, &existing); err != nil {
			return nil, fmt.Errorf("기존 리소스 삭제 실패: %w", err)
		}
		log.Printf("기존 SparkApplication 삭제됨: %s/%s", namespace, name)
	}

	// 새로 생성
	if err := k8sClient.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("SparkApplication 생성 실패: %w", err)
	}

	log.Printf("SparkApplication 생성됨: %s/%s", namespace, name)

	return &CreateResult{
		Name:      name,
		Namespace: namespace,
	}, nil
}

// CreateResult - CR 생성 결과
type CreateResult struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}
