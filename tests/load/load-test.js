import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const authLatency = new Trend('auth_latency');
const userLatency = new Trend('user_latency');

// Test configuration
export const options = {
  stages: [
    { duration: '2m', target: 10 },   // Ramp up to 10 users
    { duration: '5m', target: 10 },   // Stay at 10 users
    { duration: '2m', target: 50 },   // Ramp up to 50 users
    { duration: '5m', target: 50 },   // Stay at 50 users
    { duration: '2m', target: 100 },  // Ramp up to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests must complete below 500ms
    http_req_failed: ['rate<0.1'],    // Error rate must be less than 10%
    errors: ['rate<0.1'],             // Custom error rate
  },
};

// Base URL
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data
const testUsers = [
  { name: 'Test User 1', email: 'test1@example.com', password: 'password123' },
  { name: 'Test User 2', email: 'test2@example.com', password: 'password123' },
  { name: 'Test User 3', email: 'test3@example.com', password: 'password123' },
];

// Helper function to get random user
function getRandomUser() {
  return testUsers[Math.floor(Math.random() * testUsers.length)];
}

// Helper function to generate unique email
function generateUniqueEmail() {
  return `test${Date.now()}${Math.random().toString(36).substr(2, 5)}@example.com`;
}

// Setup function - runs once before the test
export function setup() {
  console.log('Setting up load test...');
  
  // Test health endpoint
  const healthResponse = http.get(`${BASE_URL}/health`);
  check(healthResponse, {
    'health check passed': (r) => r.status === 200,
  });
  
  return { baseUrl: BASE_URL };
}

// Main test function
export default function(data) {
  const { baseUrl } = data;
  
  // Random sleep between requests
  sleep(Math.random() * 3 + 1);
  
  // Random test scenario selection
  const scenario = Math.random();
  
  if (scenario < 0.3) {
    // 30% - Health checks
    testHealthEndpoints(baseUrl);
  } else if (scenario < 0.5) {
    // 20% - User registration
    testUserRegistration(baseUrl);
  } else if (scenario < 0.7) {
    // 20% - User authentication
    testUserAuthentication(baseUrl);
  } else if (scenario < 0.9) {
    // 20% - User CRUD operations
    testUserCRUD(baseUrl);
  } else {
    // 10% - Metrics endpoint
    testMetricsEndpoint(baseUrl);
  }
}

// Test health endpoints
function testHealthEndpoints(baseUrl) {
  const endpoints = ['/health', '/health/detailed', '/ready'];
  
  endpoints.forEach(endpoint => {
    const response = http.get(`${baseUrl}${endpoint}`);
    
    check(response, {
      [`${endpoint} status is 200`]: (r) => r.status === 200,
      [`${endpoint} response time < 200ms`]: (r) => r.timings.duration < 200,
    });
    
    if (response.status !== 200) {
      errorRate.add(1);
    }
  });
}

// Test user registration
function testUserRegistration(baseUrl) {
  const user = {
    name: `Load Test User ${Date.now()}`,
    email: generateUniqueEmail(),
    password: 'password123',
  };
  
  const startTime = Date.now();
  const response = http.post(`${baseUrl}/api/auth/register`, JSON.stringify(user), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  authLatency.add(Date.now() - startTime);
  
  check(response, {
    'registration status is 201': (r) => r.status === 201,
    'registration response has token': (r) => JSON.parse(r.body).token !== undefined,
    'registration response has user': (r) => JSON.parse(r.body).user !== undefined,
  });
  
  if (response.status !== 201) {
    errorRate.add(1);
  }
}

// Test user authentication
function testUserAuthentication(baseUrl) {
  const user = getRandomUser();
  
  const startTime = Date.now();
  const response = http.post(`${baseUrl}/api/auth/login`, JSON.stringify({
    email: user.email,
    password: user.password,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  authLatency.add(Date.now() - startTime);
  
  check(response, {
    'login status is 200': (r) => r.status === 200,
    'login response has token': (r) => JSON.parse(r.body).token !== undefined,
  });
  
  if (response.status !== 200) {
    errorRate.add(1);
  }
}

// Test user CRUD operations
function testUserCRUD(baseUrl) {
  // First, register a user to get a token
  const user = {
    name: `CRUD Test User ${Date.now()}`,
    email: generateUniqueEmail(),
    password: 'password123',
  };
  
  const registerResponse = http.post(`${baseUrl}/api/auth/register`, JSON.stringify(user), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  if (registerResponse.status !== 201) {
    errorRate.add(1);
    return;
  }
  
  const token = JSON.parse(registerResponse.body).token;
  const userId = JSON.parse(registerResponse.body).user.id;
  
  // Test GET user
  const startTime = Date.now();
  const getUserResponse = http.get(`${baseUrl}/api/users/${userId}`, {
    headers: { 'Authorization': `Bearer ${token}` },
  });
  
  userLatency.add(Date.now() - startTime);
  
  check(getUserResponse, {
    'get user status is 200': (r) => r.status === 200,
    'get user returns correct user': (r) => JSON.parse(r.body).id === userId,
  });
  
  if (getUserResponse.status !== 200) {
    errorRate.add(1);
  }
  
  // Test UPDATE user
  const updateData = {
    name: `Updated ${user.name}`,
    email: generateUniqueEmail(),
  };
  
  const updateResponse = http.put(`${baseUrl}/api/users/${userId}`, JSON.stringify(updateData), {
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  });
  
  check(updateResponse, {
    'update user status is 200': (r) => r.status === 200,
    'update user returns updated data': (r) => JSON.parse(r.body).name === updateData.name,
  });
  
  if (updateResponse.status !== 200) {
    errorRate.add(1);
  }
  
  // Test DELETE user
  const deleteResponse = http.del(`${baseUrl}/api/users/${userId}`, null, {
    headers: { 'Authorization': `Bearer ${token}` },
  });
  
  check(deleteResponse, {
    'delete user status is 204': (r) => r.status === 204,
  });
  
  if (deleteResponse.status !== 204) {
    errorRate.add(1);
  }
}

// Test metrics endpoint
function testMetricsEndpoint(baseUrl) {
  const response = http.get(`${baseUrl}/metrics`);
  
  check(response, {
    'metrics status is 200': (r) => r.status === 200,
    'metrics contains prometheus data': (r) => r.body.includes('http_requests_total'),
  });
  
  if (response.status !== 200) {
    errorRate.add(1);
  }
}

// Teardown function - runs once after the test
export function teardown(data) {
  console.log('Load test completed');
} 