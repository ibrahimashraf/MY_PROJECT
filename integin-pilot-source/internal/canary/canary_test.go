package canary

import "testing"

func TestCanaryRequiresHealthyStateBeforeTraffic(t *testing.T) {
	deployment, err := New("deploy-1", "candidate-1", "baseline-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := deployment.SetTraffic(10); err == nil {
		t.Fatal("unhealthy canary should not receive traffic")
	}
	deployment.SetHealth(HealthHealthy)
	if err := deployment.SetTraffic(10); err != nil {
		t.Fatal(err)
	}
	deployment.MarkRollbackReady(true)
	if deployment.TrafficPercent != 10 || !deployment.RollbackReady {
		t.Fatalf("unexpected deployment: %#v", deployment)
	}
	if err := deployment.SetTraffic(101); err == nil {
		t.Fatal("invalid traffic should fail")
	}
}
