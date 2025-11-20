# Development Guidelines & Code Quality Standards

## 🎯 Core Principles

### 1. **Architecture & Organization**
- **Always think architecturally** - Design before implementing
- **Proactively create new files** for distinct responsibilities
- **Logical folder structure** - Group related functionality
- **Clean separation of concerns** between layers

### 2. **File Size Limits**
- **Ideal Target**: Under 100 lines per file/component
- **Hard Limit**: Never exceed 200 lines per file
- **Split early** - Break down large files proactively

### 3. **Single Responsibility Principle (SRP)**
**Aggressively separate concerns** into distinct modules:

#### **Business Logic & State Management**
- Core business rules and workflows
- Service layer implementations
- Business state transitions

#### **Data Access & API Services**
- Database repositories and queries
- External API integrations
- Data persistence layer

#### **Utility & Helper Functions**
- Common utilities and helpers
- Shared functionality
- Helper methods and constants

#### **UI/Presentation Components**
- HTTP request/response handling
- Response formatting
- User interface logic

#### **Configuration & Constants**
- Environment configuration
- System constants
- Configuration management

#### **Data Models & Type Definitions**
- Database models
- API request/response structs
- Type definitions

## 🔧 Implementation Guidelines

### **Code Organization**
```
internal/
├── config/           # Configuration management
├── database/         # Data models and repository
├── server/          # HTTP service implementation
│   ├── billing/     # Billing-specific logic
│   ├── cart/        # Cart management
│   ├── core/        # Core business logic
│   ├── subscription/# Subscription handling
│   └── tests/       # Organized test suites
├── utils/           # Utility functions
├── webhooks/        # External webhook handling
└── testutil/        # Testing utilities
```

### **Testing Strategy**
- **Dual approach**: Mock tests + Real database integration tests
- **Organized test suites** by functionality
- **Fast development** with mocks
- **Integration validation** with real database

### **Error Handling**
- **Structured error responses** with consistent format
- **Context-aware error handling**
- **Proper error propagation** up the call stack

### **Database Design**
- **Repository pattern** for data access
- **Clean database models** with proper relationships
- **Transaction management** for complex operations
- **Connection pooling** for performance

## 📋 Quality Standards

### **Naming Conventions**
- **Clear, descriptive names** that explain purpose
- **Consistent naming patterns** across the codebase
- **Avoid abbreviations** unless widely understood

### **Documentation**
- **Self-documenting code** through clear naming
- **Comments for complex logic** only
- **API documentation** via OpenAPI specification

### **Performance**
- **Connection pooling** for database operations
- **Efficient queries** with proper indexing
- **Context-based timeouts** for all operations
- **Rate limiting** for external APIs

### **Security**
- **Environment-based secrets** management
- **Input validation** for all external data
- **SQL injection prevention** via parameterized queries
- **Webhook signature verification** for external calls

## 🚀 Development Workflow

### **Before Writing Code**
1. **Design the architecture** - Plan file structure
2. **Identify concerns** - Separate business logic from infrastructure
3. **Plan interfaces** - Define clean boundaries

### **While Writing Code**
1. **Keep files under 100 lines** - Split when approaching limit
2. **Follow SRP strictly** - One responsibility per file
3. **Write tests first** - TDD approach when feasible
4. **Document interfaces** - Clear function and struct definitions

### **Code Review Checklist**
- [ ] File size under 100 lines (hard limit: 200)
- [ ] Single responsibility clearly defined
- [ ] Clean separation from other concerns
- [ ] Proper error handling
- [ ] Test coverage where applicable
- [ ] Follows established patterns

## 🎯 Success Metrics

- **Maintainable**: Easy to understand and modify
- **Testable**: Components can be tested in isolation
- **Scalable**: Clean architecture supports growth
- **Consistent**: Uniform patterns throughout codebase
- **Production-ready**: Robust error handling and monitoring