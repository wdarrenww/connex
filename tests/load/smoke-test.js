import http from 'k6/http';
import { check } from 'k6';

// Smoke test configuration
export const options = {
  vus: 1,
  duration: '30s',
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.1'],
  },
};

// Base URL
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Main test function
export default function() {
  // Test health endpoint
  const healthResponse = http.get(`${BASE_URL}/health`);
  
  check(healthResponse, {
    'health check status is 200': (r) => r.status === 200,
    'health check response time < 200ms': (r) => r.timings.duration < 200,
  });
  
  // Test detailed health endpoint
  const detailedHealthResponse = http.get(`${BASE_URL}/health/detailed`);
  
  check(detailedHealthResponse, {
    'detailed health check status is 200': (r) => r.status === 200,
    'detailed health check response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  // Test readiness endpoint
  const readyResponse = http.get(`${BASE_URL}/ready`);
  
  check(readyResponse, {
    'readiness check status is 200': (r) => r.status === 200,
    'readiness check response time < 200ms': (r) => r.timings.duration < 200,
  });
  
  // Test user registration
  const user = {
    name: `Smoke Test User ${Date.now()}`,
    email: `smoke${Date.now()}@example.com`,
    password: 'password123',
  };
  
  const registerResponse = http.post(`${BASE_URL}/api/auth/register`, JSON.stringify(user), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  check(registerResponse, {
    'user registration status is 201': (r) => r.status === 201,
    'user registration response has token': (r) => JSON.parse(r.body).token !== undefined,
    'user registration response time < 1000ms': (r) => r.timings.duration < 1000,
  });
  
  // Test user login
  const loginResponse = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
    email: user.email,
    password: user.password,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  check(loginResponse, {
    'user login status is 200': (r) => r.status === 200,
    'user login response has token': (r) => JSON.parse(r.body).token !== undefined,
    'user login response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  // Test metrics endpoint (if accessible)
  const metricsResponse = http.get(`${BASE_URL}/metrics`);
  
  check(metricsResponse, {
    'metrics endpoint status is valid': (r) => r.status >= 200 && r.status < 500,
    'metrics endpoint response time < 500ms': (r) => r.timings.duration < 500,
  });
} 