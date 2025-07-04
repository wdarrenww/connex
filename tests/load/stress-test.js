import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('stress_errors');
const responseTime = new Trend('stress_response_time');

// Stress test configuration
export const options = {
  stages: [
    { duration: '1m', target: 50 },   // Ramp up to 50 users
    { duration: '2m', target: 100 },  // Ramp up to 100 users
    { duration: '3m', target: 200 },  // Ramp up to 200 users
    { duration: '5m', target: 200 },  // Stay at 200 users (stress)
    { duration: '2m', target: 300 },  // Ramp up to 300 users (peak stress)
    { duration: '3m', target: 300 },  // Stay at 300 users (peak stress)
    { duration: '2m', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% of requests must complete below 1s
    http_req_failed: ['rate<0.2'],     // Error rate must be less than 20%
    stress_errors: ['rate<0.2'],       // Custom error rate
  },
};

// Base URL
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data
const testUsers = Array.from({ length: 100 }, (_, i) => ({
  name: `Stress User ${i}`,
  email: `stress${i}@example.com`,
  password: 'password123',
}));

// Helper function to generate unique email
function generateUniqueEmail() {
  return `stress${Date.now()}${Math.random().toString(36).substr(2, 5)}@example.com`;
}

// Setup function
export function setup() {
  console.log('Setting up stress test...');
  
  // Pre-register some users for testing
  const users = [];
  for (let i = 0; i < 10; i++) {
    const user = {
      name: `Preload User ${i}`,
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
  
  // Minimal sleep to create maximum stress
  sleep(Math.random() * 0.5);
  
  // Random test scenario selection
  const scenario = Math.random();
  
  if (scenario < 0.4) {
    // 40% - Heavy authentication load
    testHeavyAuthLoad(baseUrl);
  } else if (scenario < 0.7) {
    // 30% - Concurrent user operations
    testConcurrentUserOps(baseUrl, users);
  } else if (scenario < 0.9) {
    // 20% - Database stress
    testDatabaseStress(baseUrl);
  } else {
    // 10% - System endpoints under stress
    testSystemEndpoints(baseUrl);
  }
}

// Test heavy authentication load
function testHeavyAuthLoad(baseUrl) {
  const user = {
    name: `Stress Auth User ${Date.now()}`,
    email: generateUniqueEmail(),
    password: 'password123',
  };
  
  const startTime = Date.now();
  
  // Register user
  const registerResponse = http.post(`${baseUrl}/api/auth/register`, JSON.stringify(user), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  responseTime.add(Date.now() - startTime);
  
  check(registerResponse, {
    'stress registration status is 201': (r) => r.status === 201,
  });
  
  if (registerResponse.status !== 201) {
    errorRate.add(1);
    return;
  }
  
  // Login immediately after registration
  const loginResponse = http.post(`${baseUrl}/api/auth/login`, JSON.stringify({
    email: user.email,
    password: user.password,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  check(loginResponse, {
    'stress login status is 200': (r) => r.status === 200,
  });
  
  if (loginResponse.status !== 200) {
    errorRate.add(1);
  }
}

// Test concurrent user operations
function testConcurrentUserOps(baseUrl, users) {
  if (users.length === 0) return;
  
  const user = users[Math.floor(Math.random() * users.length)];
  
  // Multiple concurrent operations on the same user
  const promises = [
    // Get user
    http.get(`${baseUrl}/api/users/${user.id}`, {
      headers: { 'Authorization': `Bearer ${user.token}` },
    }),
    
    // Update user
    http.put(`${baseUrl}/api/users/${user.id}`, JSON.stringify({
      name: `Updated ${user.name} ${Date.now()}`,
      email: generateUniqueEmail(),
    }), {
      headers: { 
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${user.token}`,
      },
    }),
    
    // Get user again
    http.get(`${baseUrl}/api/users/${user.id}`, {
      headers: { 'Authorization': `Bearer ${user.token}` },
    }),
  ];
  
  // Execute all requests
  const responses = promises.map(p => p);
  
  responses.forEach((response, index) => {
    check(response, {
      [`concurrent op ${index} status is valid`]: (r) => r.status >= 200 && r.status < 500,
    });
    
    if (response.status >= 400) {
      errorRate.add(1);
    }
  });
}

// Test database stress
function testDatabaseStress(baseUrl) {
  // Create many users rapidly
  const users = [];
  for (let i = 0; i < 5; i++) {
    users.push({
      name: `DB Stress User ${Date.now()}_${i}`,
      email: generateUniqueEmail(),
      password: 'password123',
    });
  }
  
  // Register all users concurrently
  const startTime = Date.now();
  const responses = users.map(user => 
    http.post(`${baseUrl}/api/auth/register`, JSON.stringify(user), {
      headers: { 'Content-Type': 'application/json' },
    })
  );
  
  responseTime.add(Date.now() - startTime);
  
  responses.forEach((response, index) => {
    check(response, {
      [`db stress registration ${index} status is 201`]: (r) => r.status === 201,
    });
    
    if (response.status !== 201) {
      errorRate.add(1);
    }
  });
  
  // Immediately try to list all users (stress the database)
  const listResponse = http.get(`${baseUrl}/api/users`, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  check(listResponse, {
    'db stress list status is valid': (r) => r.status >= 200 && r.status < 500,
  });
  
  if (listResponse.status >= 400) {
    errorRate.add(1);
  }
}

// Test system endpoints under stress
function testSystemEndpoints(baseUrl) {
  const endpoints = ['/health', '/health/detailed', '/ready', '/metrics'];
  
  // Hit all endpoints rapidly
  const startTime = Date.now();
  const responses = endpoints.map(endpoint => 
    http.get(`${baseUrl}${endpoint}`)
  );
  
  responseTime.add(Date.now() - startTime);
  
  responses.forEach((response, index) => {
    check(response, {
      [`system endpoint ${endpoints[index]} status is valid`]: (r) => r.status >= 200 && r.status < 500,
    });
    
    if (response.status >= 400) {
      errorRate.add(1);
    }
  });
}

// Teardown function
export function teardown(data) {
  console.log('Stress test completed');
} 