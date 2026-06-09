# Developing Enterprise Applications, Evolutionary and Modular Architectures (Course)

This repository contains the source code used in the course **Developing Enterprise Applications, Evolutionary and Modular Architectures**, with a focus on how to evolve a real system from a simple monolith to a modular, testable structure ready to scale.

The goal of the course is to show **how to modularize systems in practice**, without overengineering and with technical decisions guided by real-world context.

# 🚨 Read carefully!

This repository follows the chronological order of the course lessons, so the code may be incomplete or non-functional in some commits. It is recommended to follow the lessons to understand the context of each change.
Upon completing the course, the final code is complete and functional and will no longer be updated as frequently.

---

## 📚 What you will find here

Each stage of the course represents a moment of architectural evolution:

- ✅ Initial single module with simple architecture
- 🔧 Evolution to layered architectures
- 🧩 Separation by context (content, streaming, billing…)
- ⚙️ Queue integration using **BullMQ**
- 🔌 Separation of entrypoints (API / Worker)
- 🧠 AI integration via Google Gemini
- 🧭 Technical governance and modularization best practices
- 💾 TypeORM usage with managed transactions
- 🧪 End-to-end automated tests

---

## 📂 Project structure

```bash
.
├── src/io/fakeflix
│ ├── module
│ │ ├── billing
│ │ │ ├── core
│ │ │ ├── http
│ │ │ ├── integration
│ │ │ ├── persistence
│ │ ├── content
│ │ │ ├── admin
│ │ │ ├── catalog
│ │ │ ├── shared
│ │ │ ├── video-processor
│ │ ├── identity
│ │ │ ├── core
│ │ │ ├── http
│ │ │ ├── persistence
│ │ └── shared
│ │     ├── core
│ │     ├── module
```

## Main Technologies and Libraries

- **Go**: Programming language.
- **GraphQL**: Used for some parts of the API.
- **PostgreSQL**: Relational database.

## Infrastructure and DevOps

- **Docker & Compose**: Used to manage the services required for the project.

## Security

- **bcrypt**: Password hashing.
- **jsonwebtoken (JWT)**: Token-based authentication.

## Tests


## Contributing

Everyone is welcome to contribute ideas to the project—just open a pull request.

1. Create a branch for your feature/fix (`git checkout -b my-feature`).
2. Commit your changes (`git commit -m 'Add new feature'`).
3. Push to the remote (`git push origin my-feature`).
4. Open a Pull Request.
