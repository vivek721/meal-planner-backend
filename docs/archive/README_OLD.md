# Meal Planner Backend

Backend API service for the AI-Powered Meal Planner application.

## Tech Stack

- **Node.js** - Runtime environment
- **Express** - Web framework
- **TypeScript** - Type-safe JavaScript
- **Zod** - Schema validation
- **Vitest** - Testing framework

## Project Structure

```
backend/
├── src/
│   ├── controllers/     # Request handlers
│   ├── models/          # Data models
│   ├── routes/          # API routes
│   ├── middleware/      # Custom middleware
│   ├── services/        # Business logic
│   ├── config/          # Configuration files
│   ├── utils/           # Utility functions
│   ├── types/           # TypeScript type definitions
│   └── server.ts        # Application entry point
├── docs/                # Backend documentation
├── tests/               # Test files
├── .env.example         # Environment variables template
├── .gitignore
├── package.json
├── tsconfig.json
└── README.md
```

## Getting Started

### Prerequisites

- Node.js 18+ installed
- npm or yarn package manager

### Installation

```bash
# Install dependencies
npm install
```

### Environment Setup

```bash
# Copy environment variables template
cp .env.example .env

# Edit .env with your configuration
```

### Development

```bash
# Start development server with hot reload
npm run dev
```

The API will be available at `http://localhost:3001`

### Build

```bash
# Compile TypeScript to JavaScript
npm run build
```

### Production

```bash
# Run compiled code
npm start
```

### Testing

```bash
# Run tests
npm test

# Run tests with coverage
npm run test:coverage
```

## API Endpoints

### Health Check
```
GET /health
```
Returns server health status.

### API Info
```
GET /api
```
Returns API version and available endpoints.

## Planned Features

Based on the documentation in `/docs`, the following features will be implemented:

### Epic 1: Authentication & User Management
- User registration and login
- JWT-based authentication
- Password reset functionality
- User profile management

### Epic 2: Meal Planning
- Weekly meal plan CRUD operations
- Drag-and-drop meal organization
- Copy/paste meal functionality
- Clear plan operations

### Epic 3: Recipe Discovery
- Recipe search and filtering
- Recipe details and nutritional information
- Save favorite recipes
- Recipe recommendations

### Epic 4: Shopping Lists
- Auto-generate shopping lists from meal plans
- Manage shopping list items
- Mark items as purchased
- Share shopping lists

### Epic 5: User Preferences
- Dietary restrictions management
- Cuisine preferences
- Cooking skill level
- Household size settings

### Epic 6: AI Features
- AI-powered meal suggestions
- Smart recipe recommendations
- Nutritional analysis
- Ingredient substitutions

### Epic 7: Dashboard
- Weekly overview
- Nutritional summaries
- Quick actions
- Recent activity

## Development Status

### Current Status: Initial Setup
- [x] Project structure created
- [x] Basic Express server configured
- [x] TypeScript configuration
- [x] Documentation migrated
- [ ] Database setup (pending)
- [ ] Authentication implementation (pending)
- [ ] API endpoints implementation (pending)
- [ ] Testing setup (pending)

## Documentation

Detailed documentation is available in the `/docs` directory:

- **Architecture**: System design and technical decisions
- **API**: API specifications and contracts
- **Epics**: Feature breakdown and requirements
- **Planning**: Implementation roadmap
- **Testing**: Testing strategies and guidelines

## Contributing

1. Follow TypeScript best practices
2. Write tests for new features
3. Use meaningful commit messages
4. Update documentation as needed

## License

MIT
