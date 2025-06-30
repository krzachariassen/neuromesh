Every time you learn something, add it to your memory so you never forget it.

Always use the TDD Enforcement Protocol for all code changes.
TDD Checklist (I will follow this religiously):
 - RED: Write failing tests that expose design flaws
 - GREEN: Write minimal code to make tests pass
 - REFACTOR: Clean up while keeping tests green
 - VALIDATE: Run all tests to ensure correctness
 - REPEAT: Never skip the cycle

When ever we work with AI, never mock the AI provide in any tests, always use the real AI provider as we can't mock the AI's behavior accurately. This ensures that our tests reflect real-world scenarios and the AI's actual performance.

Always apply SOLID principles
 - Single Responsibility Principle: A class should have a single responsibility
 - Open-Closed Principle: Classes open for extension, closed for modification
 - Liskov Substitution Principle: Subtypes must be substitutable for base types
 - Interface Segregation Principle: Clients should not depend on unused interfaces
 - Dependency Inversion Principle: Depend on abstractions, not concrete implementations

Always Apply clean architecture principles:
- Use interfaces to define boundaries
- Keep business logic separate from infrastructure
- Use dependency injection to manage dependencies
- Ensure high cohesion and low coupling
- Use SOLID principles to guide design decisions
- Write tests first, then implement functionality
- Use descriptive names for functions and variables
- Keep functions small and focused
- Avoid side effects in functions
- Use exceptions for error handling, not return codes
- Use meaningful comments to explain complex logic
- us git braches for features and bug fixes
- Use descriptive commit messages
- Use version control for all code changes
- Use pull requests for code reviews

Always apply YAGNI principles:
- You Aren't Gonna Need It: Don't add functionality until it's necessary
- Focus on current requirements, not future possibilities
- Avoid speculative generality
- Keep code simple and focused on current needs
- Refactor only when necessary, not preemptively
- Write tests for current functionality, not future features