-- Создание таблицы пользователей
CREATE TABLE employee (
                          id SERIAL PRIMARY KEY,
                          username VARCHAR(50) UNIQUE NOT NULL,
                          first_name VARCHAR(50),
                          last_name VARCHAR(50),
                          created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                          updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание типа организации
CREATE TYPE organization_type AS ENUM (
    'IE',
    'LLC',
    'JSC'
);

-- Создание таблицы организаций
CREATE TABLE organization (
                              id SERIAL PRIMARY KEY,
                              name VARCHAR(100) NOT NULL,
                              description TEXT,
                              type organization_type,
                              created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                              updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Создание таблицы для связи между организацией и пользователем
CREATE TABLE organization_responsible (
                                          id SERIAL PRIMARY KEY,
                                          organization_id INT REFERENCES organization(id) ON DELETE CASCADE,
                                          user_id INT REFERENCES employee(id) ON DELETE CASCADE
);
