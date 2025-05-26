# Medical Record Transfer Microservice

## Project Goals / Objetivos do Projeto

This microservice provides a secure and efficient solution for importing, exporting, and transferring complete medical records (including patient history and treatment data) between healthcare facilities. It aims to ensure uninterrupted patient care during transfers while maintaining strict data integrity.

*PT-BR:* Este microsserviço fornece uma solução segura e eficiente para importar, exportar e transferir prontuários médicos completos (incluindo histórico do paciente e dados de tratamento) entre unidades de saúde. Visa garantir o cuidado ininterrupto do paciente durante as transferências, mantendo a estrita integridade dos dados.

## Solution Architecture / Arquitetura da Solução

The system is designed as a microservice built with Go, utilizing the Fiber framework for its API layer and PostgreSQL (via Supabase) for data persistence. The architecture emphasizes security, data integrity, and a reactive design following SOLID principles. It is containerized using Docker and Docker Compose, making it Kubernetes-ready for scalable deployments. Comprehensive testing (unit and integration) and CI/CD via GitHub Actions are integral parts of the development lifecycle.

*PT-BR:* O sistema é projetado como um microsserviço construído em Go, utilizando o framework Fiber para sua camada de API e PostgreSQL (via Supabase) para persistência de dados. A arquitetura enfatiza segurança, integridade dos dados e um design reativo seguindo os princípios SOLID. É conteinerizado usando Docker e Docker Compose, tornando-o pronto para Kubernetes para implantações escaláveis. Testes abrangentes (unitários e de integração) e CI/CD via GitHub Actions são partes integrantes do ciclo de vida de desenvolvimento.

## Microservice Objectives / Objetivos do Microsserviço

The primary objectives of this microservice are:

*   **Secure Medical Record Transfers**: Enable the secure transfer of complete patient histories and treatment data between different healthcare facilities.
*   **Data Integrity**: Maintain the accuracy and consistency of medical records throughout the transfer process.
*   **Uninterrupted Patient Care**: Facilitate seamless transitions of patient information to prevent disruptions in care.
*   **Reactive and SOLID Architecture**: Implement a system that is responsive and adheres to SOLID principles for maintainability and scalability.
*   **Comprehensive Testing**: Ensure reliability through extensive unit and integration testing.
*   **Containerized Deployment**: Provide a Dockerized application, ready for orchestration with Kubernetes.
*   **Audit Logging**: Implement audit logging for all transfer operations to ensure traceability and accountability.
*   **Role-Based Access Control (RBAC)**: Secure sensitive operations through a robust RBAC mechanism.

*PT-BR:* Os objetivos principais deste microsserviço são:

*   **Transferências Seguras de Prontuários Médicos**: Permitir a transferência segura de históricos completos de pacientes e dados de tratamento entre diferentes unidades de saúde.
*   **Integridade dos Dados**: Manter a precisão e consistência dos prontuários médicos durante todo o processo de transferência.
*   **Cuidado Ininterrupto ao Paciente**: Facilitar transições suaves de informações do paciente para evitar interrupções no atendimento.
*   **Arquitetura Reativa e SOLID**: Implementar um sistema responsivo e que adere aos princípios SOLID para manutenibilidade e escalabilidade.
*   **Testes Abrangentes**: Assegurar a confiabilidade através de extensivos testes unitários e de integração.
*   **Implantação Conteinerizada**: Fornecer uma aplicação Dockerizada, pronta para orquestração com Kubernetes.
*   **Registro de Auditoria**: Implementar registro de auditoria para todas as operações de transferência para garantir rastreabilidade e responsabilidade.
*   **Controle de Acesso Baseado em Função (RBAC)**: Proteger operações sensíveis através de um mecanismo RBAC robusto.

---

### Technical Stack / Stack Tecnológico

*   **Language / Linguagem**: Go 1.24
*   **Framework / Framework**: Fiber (v2)
*   **API Documentation / Documentação da API**: Swagger UI
*   **Database / Banco de Dados**: PostgreSQL (Supabase)
*   **Testing / Testes**: Testify + GoMock
*   **Containerization / Conteinerização**: Docker + Docker Compose
*   **CI/CD**: GitHub Actions (sample included)

### Development Principles / Princípios de Desenvolvimento

*   SOLID Compliance
*   Comprehensive Testing Strategy
*   Security First

### Roadmap / Roteiro

*   Implement HL7 FHIR compatibility
*   Add GraphQL alternative endpoint
*   Develop patient consent management
*   Create dashboard for transfer monitoring
