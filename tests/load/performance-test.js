import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('performance_errors');
const authLatency = new Trend('auth_performance');
const userLatency = new Trend('user_performance');
const dbLatency = new Trend('db_performance');
const cacheLatency = new Trend('cache_performance');
const requestCounter = new Counter('total_requests');

// Performance test configuration
export const options = {
  stages: [
    { duration: '1m', target: 10 },   // Warm up
    { duration: '2m', target: 10 },   // Baseline
    { duration: '2m', target: 50 },   // Medium load
    { duration: '2m', target: 100 },  // High load
    { duration: '1m', target: 0 },    // Cool down
  ],
  thresholds: {
    http_req_duration: ['p(50)<200', 'p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.05'],
    auth_performance: ['p(95)<300'],
    user_performance: ['p(95)<400'],
    db_performance: ['p(95)<200'],
    cache_performance: ['p(95)<50'],
  },
};

// Base URL
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Helper function to generate unique email
function generateUniqueEmail() {
  return `perf${Date.now()}${Math.random().toString(36).substr(2, 5)}@example.com`;
}

// Setup function
export function setup() {
  console.log('Setting up performance test...');
  
  // Create test users for performance testing
  const users = [];
  for (let i = 0; i < 20; i++) {
    const user = {
      name: `Performance User ${i}`,
      email: generateUniqueEmail(),
      password: 'password123',
    };
    
    const response = http.post(`${BASE_URL}/api/auth/register`, JSON.stringify(user), {
      headers: { 'Content-Type': 'application/json' },
    });
    
    if (response.status === 201) {
      const responseData = JSON.parse(response.body);
      users.push({
        ...user,
        token: responseData.token,
        id: responseData.user.id,
      });
    }
  }
  
  return { baseUrl: BASE_URL, users };
}

// Main test function
export default function(data) {
  const { baseUrl, users } = data;
  
  // Random sleep to simulate real user behavior
  sleep(Math.random() * 2 + 0.5);
  
  // Test scenario distribution
  const scenario = Math.random();
  
  if (scenario < 0.25) {
    // 25% - Authentication performance
    testAuthPerformance(baseUrl);
  } else if (scenario < 0.45) {
    // 20% - User CRUD performance
    testUserPerformance(baseUrl, users);
  } else if (scenario < 0.65) {
    // 20% - Database performance
    testDatabasePerformance(baseUrl);
  } else if (scenario < 0.85) {
    // 20% - Cache performance
    testCachePerformance(baseUrl, users);
  } else {
    // 15% - System performance
    testSystemPerformance(baseUrl);
  }
}

// Test authentication performance
function testAuthPerformance(baseUrl) {
  const user = {
    name: `Perf Auth User ${Date.now()}`,
    email: generateUniqueEmail(),
    password: 'password123',
  };
  
  // Test registration performance
  const startTime = Date.now();
  const registerResponse = http.post(`${baseUrl}/api/auth/register`, JSON.stringify(user), {
    headers: { 'Content-Type': 'application/json' },
  });
  authLatency.add(Date.now() - startTime);
  requestCounter.add(1);
  
  check(registerResponse, {
    'auth registration performance status is 201': (r) => r.status === 201,
    'auth registration performance time < 500ms': (r) => r.timings.duration < 500,
  });
  
  if (registerResponse.status !== 201) {
    errorRate.add(1);
    return;
  }
  
  // Test login performance
  const loginStartTime = Date.now();
  const loginResponse = http.post(`${baseUrl}/api/auth/login`, JSON.stringify({
    email: user.email,
    password: user.password,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  authLatency.add(Date.now() - loginStartTime);
  requestCounter.add(1);
  
  check(loginResponse, {
    'auth login performance status is 200': (r) => r.status === 200,
    'auth login performance time < 300ms': (r) => r.timings.duration < 300,
  });
  
  if (loginResponse.status !== 200) {
    errorRate.add(1);
  }
}

// Test user CRUD performance
function testUserPerformance(baseUrl, users) {
  if (users.length === 0) return;
  
  const user = users[Math.floor(Math.random() * users.length)];
  
  // Test GET user performance
  const getStartTime = Date.now();
  const getResponse = http.get(`${baseUrl}/api/users/${user.id}`, {
    headers: { 'Authorization': `Bearer ${user.token}` },
  });
  userLatency.add(Date.now() - getStartTime);
  requestCounter.add(1);
  
  check(getResponse, {
    'user get performance status is 200': (r) => r.status === 200,
    'user get performance time < 200ms': (r) => r.timings.duration < 200,
  });
  
  if (getResponse.status !== 200) {
    errorRate.add(1);
  }
  
  // Test UPDATE user performance
  const updateStartTime = Date.now();
  const updateResponse = http.put(`${baseUrl}/api/users/${user.id}`, JSON.stringify({
    name: `Updated ${user.name} ${Date.now()}`,
    email: generateUniqueEmail(),
  }), {
    headers: { 
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${user.token}`,
    },
  });
  userLatency.add(Date.now() - updateStartTime);
  requestCounter.add(1);
  
  check(updateResponse, {
    'user update performance status is 200': (r) => r.status === 200,
    'user update performance time < 400ms': (r) => r.timings.duration < 400,
  });
  
  if (updateResponse.status !== 200) {
    errorRate.add(1);
  }
}

// Test database performance
function testDatabasePerformance(baseUrl) {
  // Test user listing performance (database intensive)
  const startTime = Date.now();
  const listResponse = http.get(`${baseUrl}/api/users`);
  dbLatency.add(Date.now() - startTime);
  requestCounter.add(1);
  
  check(listResponse, {
    'db list performance status is valid': (r) => r.status >= 200 && r.status < 500,
    'db list performance time < 200ms': (r) => r.timings.duration < 200,
  });
  
  if (listResponse.status >= 400) {
    errorRate.add(1);
  }
  
  // Test user creation performance
  const user = {
    name: `DB Perf User ${Date.now()}`,
    email: generateUniqueEmail(),
    password: 'password123',
  };
  
  const createStartTime = Date.now();
  const createResponse = http.post(`${baseUrl}/api/auth/register`, JSON.stringify(user), {
    headers: { 'Content-Type': 'application/json' },
  });
  dbLatency.add(Date.now() - createStartTime);
  requestCounter.add(1);
  
  check(createResponse, {
    'db create performance status is 201': (r) => r.status === 201,
    'db create performance time < 300ms': (r) => r.timings.duration < 300,
  });
  
  if (createResponse.status !== 201) {
    errorRate.add(1);
  }
}

// Test cache performance
function testCachePerformance(baseUrl, users) {
  if (users.length === 0) return;
  
  const user = users[Math.floor(Math.random() * users.length)];
  
  // Test cached GET requests (should be faster on subsequent calls)
  const requests = [];
  
  for (let i = 0; i < 3; i++) {
    const startTime = Date.now();
    const response = http.get(`${baseUrl}/api/users/${user.id}`, {
      headers: { 'Authorization': `Bearer ${user.token}` },
    });
    cacheLatency.add(Date.now() - startTime);
    requestCounter.add(1);
    requests.push(response);
    
    // Small delay between requests
    sleep(0.1);
  }
  
  requests.forEach((response, index) => {
    check(response, {
      [`cache request ${index} performance status is 200`]: (r) => r.status === 200,
      [`cache request ${index} performance time < 100ms`]: (r) => r.timings.duration < 100,
    });
    
    if (response.status !== 200) {
      errorRate.add(1);
    }
  });
}

// Test system performance
function testSystemPerformance(baseUrl) {
  const endpoints = ['/health', '/health/detailed', '/ready', '/metrics'];
  
  endpoints.forEach(endpoint => {
    const startTime = Date.now();
    const response = http.get(`${baseUrl}${endpoint}`);
    const duration = Date.now() - startTime;
    requestCounter.add(1);
    
    check(response, {
      [`system ${endpoint} performance status is valid`]: (r) => r.status >= 200 && r.status < 500,
      [`system ${endpoint} performance time < 100ms`]: (r) => r.timings.duration < 100,
    });
    
    if (response.status >= 400) {
      errorRate.add(1);
    }
  });
}

// Teardown function
export function teardown(data) {
  console.log('Performance test completed');
  console.log(`Total requests made: ${requestCounter.count}`);
} 