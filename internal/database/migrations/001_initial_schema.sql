-- +goose Up

-- Аккаунты с MTLAP или MTLAC
CREATE TABLE accounts (
    account_id TEXT PRIMARY KEY,

    -- Балансы ключевых токенов
    mtlap_balance NUMERIC(20, 7) DEFAULT 0,
    mtlac_balance NUMERIC(20, 7) DEFAULT 0,
    native_balance NUMERIC(20, 7) DEFAULT 0,

    -- Расчётная стоимость всех активов в XLM
    total_xlm_value NUMERIC(20, 7) DEFAULT 0,

    -- Делегации
    delegate_to TEXT,
    is_council_ready BOOLEAN DEFAULT FALSE,
    received_votes INTEGER DEFAULT 0,

    -- Флаги ошибок делегации
    has_delegation_error BOOLEAN DEFAULT FALSE,
    has_cycle_error BOOLEAN DEFAULT FALSE,
    cycle_path TEXT[],

    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_accounts_mtlap ON accounts(mtlap_balance DESC) WHERE mtlap_balance > 0;
CREATE INDEX idx_accounts_mtlac ON accounts(mtlac_balance DESC) WHERE mtlac_balance > 0;
CREATE INDEX idx_accounts_votes ON accounts(received_votes DESC) WHERE is_council_ready = TRUE;
CREATE INDEX idx_accounts_xlm_value ON accounts(total_xlm_value DESC) WHERE mtlac_balance > 0;

-- Все балансы аккаунта
CREATE TABLE account_balances (
    account_id TEXT NOT NULL,
    asset_code TEXT NOT NULL,
    asset_issuer TEXT NOT NULL DEFAULT '',
    balance NUMERIC(20, 7) NOT NULL,
    xlm_value NUMERIC(20, 7),

    PRIMARY KEY (account_id, asset_code, asset_issuer)
);

CREATE INDEX idx_account_balances_account ON account_balances(account_id);

-- ManageData аккаунта
CREATE TABLE account_metadata (
    account_id TEXT NOT NULL,
    data_key TEXT NOT NULL,
    data_index TEXT NOT NULL,
    data_value TEXT NOT NULL,

    PRIMARY KEY (account_id, data_key, data_index)
);

CREATE INDEX idx_account_metadata_key ON account_metadata(data_key);
CREATE INDEX idx_account_metadata_account ON account_metadata(account_id);

-- Настройки типов связей
CREATE TABLE relation_type_settings (
    relation_type TEXT PRIMARY KEY,
    paired_with TEXT,
    requires_confirmation BOOLEAN DEFAULT FALSE,
    description TEXT
);

INSERT INTO relation_type_settings (relation_type, paired_with, requires_confirmation, description) VALUES
-- Участие в организациях
('MyPart', 'PartOf', TRUE, 'Организация заявляет участника'),
('PartOf', 'MyPart', TRUE, 'Участник подтверждает членство'),
('RecommendToMTLA', NULL, FALSE, 'Рекомендация в Ассоциацию'),

-- Личные отношения
('OneFamily', 'OneFamily', TRUE, 'Ближайшая родственная связь'),
('Spouse', 'Spouse', TRUE, 'Фактический супруг'),
('Guardian', 'Ward', TRUE, 'Опекун'),
('Ward', 'Guardian', TRUE, 'Опекаемый'),
('Sympathy', NULL, FALSE, 'Романтическая привязанность'),
('Love', NULL, FALSE, 'Крайняя форма симпатии'),
('Divorce', NULL, FALSE, 'Потеря статуса брака/опекунства'),

-- Личный рейтинг (кредитная оценка)
('A', NULL, FALSE, 'Надёжный проверенный субъект (1000+ EURMTL)'),
('B', NULL, FALSE, 'Кредитоспособное лицо (до 1000 EURMTL)'),
('C', NULL, FALSE, 'Слабая претензия по обязательствам'),
('D', NULL, FALSE, 'Сильное дефолтное состояние (1000+ EURMTL)'),

-- Сотрудничество
('Employer', 'Employee', TRUE, 'Работодатель'),
('Employee', 'Employer', TRUE, 'Сотрудник'),
('Contractor', 'Client', TRUE, 'Подрядчик'),
('Client', 'Contractor', TRUE, 'Заказчик'),
('Partnership', 'Partnership', TRUE, 'Долгосрочные партнерские отношения'),
('Collaboration', 'Collaboration', TRUE, 'Совместная работа с равноправным вкладом'),

-- Имущественная связь
('OwnershipFull', 'Owner', TRUE, 'Владение счётом на 95%+'),
('OwnershipMajority', 'OwnerMajority', TRUE, 'Владение счётом на 25-95%'),
('OwnershipMinority', 'OwnerMinority', TRUE, 'Владение счётом менее 25%'),
('Owner', 'OwnershipFull', TRUE, 'Счёт владеет аккаунтом на 95%+'),
('OwnerMajority', 'OwnershipMajority', TRUE, 'Счёт владеет аккаунтом на 25-95%'),
('OwnerMinority', 'OwnershipMinority', TRUE, 'Счёт владеет аккаунтом менее 25%'),

-- Прочее
('WelcomeGuest', NULL, FALSE, 'Приглашение в гости'),
('FactionMember', NULL, FALSE, 'Отношение к фракции Ассоциации');

-- Связи между аккаунтами
CREATE TABLE relationships (
    source_account_id TEXT NOT NULL,
    target_account_id TEXT NOT NULL,
    relation_type TEXT NOT NULL,
    relation_index TEXT NOT NULL,

    PRIMARY KEY (source_account_id, relation_type, relation_index)
);

CREATE INDEX idx_relationships_target ON relationships(target_account_id);
CREATE INDEX idx_relationships_type ON relationships(relation_type);
CREATE INDEX idx_relationships_type_target ON relationships(relation_type, target_account_id);
CREATE INDEX idx_relationships_source ON relationships(source_account_id);

-- View для подтверждённых связей (обе стороны установили парные теги)
CREATE VIEW confirmed_relationships AS
SELECT
    r1.source_account_id,
    r1.target_account_id,
    r1.relation_type,
    r1.relation_index AS source_index,
    r2.relation_index AS target_index,
    s.description
FROM relationships r1
JOIN relation_type_settings s ON r1.relation_type = s.relation_type
JOIN relationships r2
    ON r1.target_account_id = r2.source_account_id
    AND r1.source_account_id = r2.target_account_id
    AND r2.relation_type = s.paired_with
WHERE s.requires_confirmation = TRUE
  AND s.paired_with IS NOT NULL;

-- Кэш цен токенов
CREATE TABLE token_prices (
    asset_code TEXT NOT NULL,
    asset_issuer TEXT NOT NULL,
    xlm_price NUMERIC(20, 7) NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (asset_code, asset_issuer)
);

-- Настройки тегов Ассоциации
CREATE TABLE association_tag_settings (
    tag_name TEXT PRIMARY KEY,
    description TEXT
);

INSERT INTO association_tag_settings (tag_name, description) VALUES
('Program', 'Постоянные программы Ассоциации'),
('Faction', 'Фракции Ассоциации');

-- Теги от Ассоциации
CREATE TABLE association_tags (
    tag_name TEXT NOT NULL,
    tag_index TEXT NOT NULL,
    target_account_id TEXT NOT NULL,

    PRIMARY KEY (tag_name, tag_index)
);

CREATE INDEX idx_association_tags_target ON association_tags(target_account_id);
CREATE INDEX idx_association_tags_name ON association_tags(tag_name);

-- +goose Down

DROP VIEW IF EXISTS confirmed_relationships;
DROP TABLE IF EXISTS association_tags;
DROP TABLE IF EXISTS association_tag_settings;
DROP TABLE IF EXISTS token_prices;
DROP TABLE IF EXISTS relationships;
DROP TABLE IF EXISTS relation_type_settings;
DROP TABLE IF EXISTS account_metadata;
DROP TABLE IF EXISTS account_balances;
DROP TABLE IF EXISTS accounts;
