-- Demo Seed Data: Kikinda Pilot
-- Vertikala: Lokalna ustanova -> Okrug -> Pokrajina -> Ministarstvo -> Vlada

-- Clear existing data (for re-runs) - only if tables exist
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'identity' AND table_name = 'agencies') THEN
        TRUNCATE identity.agencies CASCADE;
    END IF;
END $$;

-- ============================================
-- AGENCIJE - Vertikalna struktura
-- ============================================

-- NIVO 1: Vlada Republike Srbije
INSERT INTO identity.agencies (id, code, name, type, status, address_street, address_city, address_postal_code, address_country, address_lat, address_lng, contact_email, contact_phone, contact_mobile, created_at, updated_at) VALUES
('a0000000-0000-0000-0000-000000000001', 'GOV-RS', 'Vlada Republike Srbije', 'GOVERNMENT', 'active', 'Trg slobode 1', 'Beograd', '11000', 'RS', 44.8176, 20.4633, 'kabinet@gov.rs', '+381 11 3617 000', '', NOW(), NOW());

-- NIVO 2: Ministarstva
INSERT INTO identity.agencies (id, code, name, type, parent_id, status, address_street, address_city, address_postal_code, address_country, address_lat, address_lng, contact_email, contact_phone, contact_mobile, created_at, updated_at) VALUES
('a0000000-0000-0000-0000-000000000010', 'MIN-SOCIAL', 'Ministarstvo za brigu o porodici i demografiju', 'MINISTRY', 'a0000000-0000-0000-0000-000000000001', 'active', 'Nemanjina 22-26', 'Beograd', '11000', 'RS', 44.8022, 20.4664, 'kabinet@minbpd.gov.rs', '+381 11 363 1461', '', NOW(), NOW()),
('a0000000-0000-0000-0000-000000000011', 'MIN-HEALTH', 'Ministarstvo zdravlja', 'MINISTRY', 'a0000000-0000-0000-0000-000000000001', 'active', 'Nemanjina 22-26', 'Beograd', '11000', 'RS', 44.8022, 20.4664, 'kabinet@zdravlje.gov.rs', '+381 11 363 1504', '', NOW(), NOW()),
('a0000000-0000-0000-0000-000000000012', 'MIN-INTERIOR', 'Ministarstvo unutrasnjih poslova', 'MINISTRY', 'a0000000-0000-0000-0000-000000000001', 'active', 'Kneza Milosa 101', 'Beograd', '11000', 'RS', 44.7950, 20.4680, 'kabinet@mup.gov.rs', '+381 11 306 2000', '', NOW(), NOW());

-- NIVO 3: Pokrajina (AP Vojvodina)
INSERT INTO identity.agencies (id, code, name, type, parent_id, status, address_street, address_city, address_postal_code, address_country, address_lat, address_lng, contact_email, contact_phone, contact_mobile, created_at, updated_at) VALUES
('a0000000-0000-0000-0000-000000000020', 'APV-SOCIAL', 'Pokrajinski sekretarijat za socijalnu politiku', 'REGIONAL', 'a0000000-0000-0000-0000-000000000010', 'active', 'Bulevar Mihajla Pupina 16', 'Novi Sad', '21000', 'RS', 45.2671, 19.8335, 'socijalna@vojvodina.gov.rs', '+381 21 487 4283', '', NOW(), NOW()),
('a0000000-0000-0000-0000-000000000021', 'APV-HEALTH', 'Pokrajinski sekretarijat za zdravstvo', 'REGIONAL', 'a0000000-0000-0000-0000-000000000011', 'active', 'Bulevar Mihajla Pupina 16', 'Novi Sad', '21000', 'RS', 45.2671, 19.8335, 'zdravstvo@vojvodina.gov.rs', '+381 21 487 4267', '', NOW(), NOW());

-- NIVO 4: Okrug (Severnobanatski okrug)
INSERT INTO identity.agencies (id, code, name, type, parent_id, status, address_street, address_city, address_postal_code, address_country, address_lat, address_lng, contact_email, contact_phone, contact_mobile, created_at, updated_at) VALUES
('a0000000-0000-0000-0000-000000000030', 'SBO-ADMIN', 'Severnobanatski upravni okrug', 'DISTRICT', 'a0000000-0000-0000-0000-000000000020', 'active', 'Trg srpskih dobrovoljaca 2', 'Kikinda', '23300', 'RS', 45.8265, 20.4681, 'okrug@kikinda.gov.rs', '+381 230 21 111', '', NOW(), NOW());

-- NIVO 5: Lokalne ustanove - KIKINDA (Pilot)
INSERT INTO identity.agencies (id, code, name, type, parent_id, status, address_street, address_city, address_postal_code, address_country, address_lat, address_lng, contact_email, contact_phone, contact_mobile, created_at, updated_at) VALUES
-- Centar za socijalni rad
('a0000000-0000-0000-0000-000000000100', 'CSR-KI', 'Centar za socijalni rad Kikinda', 'SOCIAL_WELFARE', 'a0000000-0000-0000-0000-000000000030', 'active', 'Trg srpskih dobrovoljaca 8', 'Kikinda', '23300', 'RS', 45.8257, 20.4678, 'csr@kikinda.org.rs', '+381 230 421 433', '', NOW(), NOW()),
-- Zdravstvene ustanove
('a0000000-0000-0000-0000-000000000101', 'OB-KI', 'Opsta bolnica Kikinda', 'HEALTHCARE', 'a0000000-0000-0000-0000-000000000021', 'active', 'Djure Jaksica 110', 'Kikinda', '23300', 'RS', 45.8300, 20.4700, 'info@bolnica-kikinda.rs', '+381 230 21 255', '', NOW(), NOW()),
('a0000000-0000-0000-0000-000000000102', 'DZ-KI', 'Dom zdravlja Kikinda', 'HEALTHCARE', 'a0000000-0000-0000-0000-000000000021', 'active', 'Jovana Jovanovica Zmaja 12', 'Kikinda', '23300', 'RS', 45.8280, 20.4650, 'info@dzkikinda.rs', '+381 230 421 555', '', NOW(), NOW()),
-- Policija
('a0000000-0000-0000-0000-000000000103', 'PU-KI', 'Policijska uprava Kikinda', 'POLICE', 'a0000000-0000-0000-0000-000000000012', 'active', 'Svetosavska 55', 'Kikinda', '23300', 'RS', 45.8250, 20.4660, 'pu-kikinda@mup.gov.rs', '+381 230 400 100', '', NOW(), NOW()),
-- Obrazovanje
('a0000000-0000-0000-0000-000000000104', 'SKOLA-KI', 'Centar za obrazovanje - Kikinda', 'EDUCATION', 'a0000000-0000-0000-0000-000000000030', 'active', 'Nemanjina 28', 'Kikinda', '23300', 'RS', 45.8240, 20.4640, 'skola@kikinda.edu.rs', '+381 230 22 564', '', NOW(), NOW()),
-- Gerontoloski centar
('a0000000-0000-0000-0000-000000000105', 'GC-KI', 'Gerontoloski centar Kikinda', 'SOCIAL_WELFARE', 'a0000000-0000-0000-0000-000000000030', 'active', 'Nemanjina 2', 'Kikinda', '23300', 'RS', 45.8230, 20.4630, 'info@gc-kikinda.rs', '+381 230 22 145', '', NOW(), NOW());

-- Susedne opstine u okrugu
INSERT INTO identity.agencies (id, code, name, type, parent_id, status, address_street, address_city, address_postal_code, address_country, address_lat, address_lng, contact_email, contact_phone, contact_mobile, created_at, updated_at) VALUES
('a0000000-0000-0000-0000-000000000110', 'CSR-SE', 'Centar za socijalni rad Senta', 'SOCIAL_WELFARE', 'a0000000-0000-0000-0000-000000000030', 'active', 'Glavni trg 1', 'Senta', '24400', 'RS', 45.9300, 20.0800, 'csr@senta.org.rs', '+381 24 811 055', '', NOW(), NOW()),
('a0000000-0000-0000-0000-000000000111', 'DZ-SE', 'Dom zdravlja Senta', 'HEALTHCARE', 'a0000000-0000-0000-0000-000000000021', 'active', 'Potviska 1', 'Senta', '24400', 'RS', 45.9310, 20.0810, 'info@dzsenta.rs', '+381 24 812 200', '', NOW(), NOW()),
('a0000000-0000-0000-0000-000000000120', 'CSR-KA', 'Centar za socijalni rad Kanjiza', 'SOCIAL_WELFARE', 'a0000000-0000-0000-0000-000000000030', 'active', 'Trg oslobodjenja 5', 'Kanjiza', '24420', 'RS', 46.0700, 20.0500, 'csr@kanjiza.org.rs', '+381 24 873 170', '', NOW(), NOW());

-- ============================================
-- RADNICI
-- ============================================

-- CSR Kikinda
INSERT INTO identity.workers (id, agency_id, employee_id, email, first_name, last_name, position, status, created_at, updated_at) VALUES
('b0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000100', 'CSR-KI-001', 'direktor@csr-kikinda.gov.rs', 'Marija', 'Nikolic', 'Direktor', 'active', NOW(), NOW()),
('b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000100', 'CSR-KI-002', 'voditelj1@csr-kikinda.gov.rs', 'Jovana', 'Petrovic', 'Voditelj slucaja', 'active', NOW(), NOW()),
('b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000100', 'CSR-KI-003', 'radnik1@csr-kikinda.gov.rs', 'Milica', 'Jovanovic', 'Socijalni radnik', 'active', NOW(), NOW()),
('b0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000100', 'CSR-KI-004', 'radnik2@csr-kikinda.gov.rs', 'Stefan', 'Markovic', 'Socijalni radnik', 'active', NOW(), NOW()),
('b0000000-0000-0000-0000-000000000005', 'a0000000-0000-0000-0000-000000000100', 'CSR-KI-005', 'psiholog@csr-kikinda.gov.rs', 'Ana', 'Stojanovic', 'Psiholog', 'active', NOW(), NOW());

-- Dom zdravlja Kikinda
INSERT INTO identity.workers (id, agency_id, employee_id, email, first_name, last_name, position, status, created_at, updated_at) VALUES
('b0000000-0000-0000-0000-000000000010', 'a0000000-0000-0000-0000-000000000102', 'DZ-KI-001', 'direktor@dz-kikinda.gov.rs', 'Dragan', 'Ilic', 'Direktor', 'active', NOW(), NOW()),
('b0000000-0000-0000-0000-000000000011', 'a0000000-0000-0000-0000-000000000102', 'DZ-KI-002', 'lekar1@dz-kikinda.gov.rs', 'Jelena', 'Todorovic', 'Lekar opste prakse', 'active', NOW(), NOW()),
('b0000000-0000-0000-0000-000000000012', 'a0000000-0000-0000-0000-000000000102', 'DZ-KI-003', 'patronaza@dz-kikinda.gov.rs', 'Snezana', 'Pavlovic', 'Patronazna sestra', 'active', NOW(), NOW());

-- Opsta bolnica Kikinda
INSERT INTO identity.workers (id, agency_id, employee_id, email, first_name, last_name, position, status, created_at, updated_at) VALUES
('b0000000-0000-0000-0000-000000000020', 'a0000000-0000-0000-0000-000000000101', 'OB-KI-001', 'direktor@ob-kikinda.gov.rs', 'Nikola', 'Djordjevic', 'Direktor', 'active', NOW(), NOW()),
('b0000000-0000-0000-0000-000000000021', 'a0000000-0000-0000-0000-000000000101', 'OB-KI-002', 'psihijatar@ob-kikinda.gov.rs', 'Maja', 'Kostic', 'Psihijatar', 'active', NOW(), NOW());

-- Gerontoloski centar
INSERT INTO identity.workers (id, agency_id, employee_id, email, first_name, last_name, position, status, created_at, updated_at) VALUES
('b0000000-0000-0000-0000-000000000030', 'a0000000-0000-0000-0000-000000000105', 'GC-KI-001', 'direktor@gc-kikinda.gov.rs', 'Gordana', 'Stankovic', 'Direktor', 'active', NOW(), NOW()),
('b0000000-0000-0000-0000-000000000031', 'a0000000-0000-0000-0000-000000000105', 'GC-KI-002', 'socradnik@gc-kikinda.gov.rs', 'Ivana', 'Ristic', 'Socijalni radnik', 'active', NOW(), NOW());

-- Ministarstvo
INSERT INTO identity.workers (id, agency_id, employee_id, email, first_name, last_name, position, status, created_at, updated_at) VALUES
('b0000000-0000-0000-0000-000000000040', 'a0000000-0000-0000-0000-000000000010', 'MIN-001', 'kabinet@minbpd.gov.rs', 'Vesna', 'Mitrovic', 'Sekretar', 'active', NOW(), NOW());

-- ============================================
-- SLUCAJEVI
-- ============================================

-- Slucaj 1: Zastita deteta - multisektorska koordinacija
INSERT INTO cases.cases (id, case_number, type, status, priority, title, description, owning_agency_id, lead_worker_id, created_at, updated_at) VALUES
('c0000000-0000-0000-0000-000000000001', 'KI-2024-001', 'CHILD_PROTECTION', 'in_progress', 'high',
 'Zastita maloletnog lica - zanemarivanje',
 'Prijava skole o sumnji na zanemarivanje deteta. Potrebna koordinacija CSR, DZ i skole.',
 'a0000000-0000-0000-0000-000000000100', 'b0000000-0000-0000-0000-000000000003',
 NOW() - INTERVAL '14 days', NOW());

-- Slucaj 2: Starija osoba - potrebna nega
INSERT INTO cases.cases (id, case_number, type, status, priority, title, description, owning_agency_id, lead_worker_id, created_at, updated_at) VALUES
('c0000000-0000-0000-0000-000000000002', 'KI-2024-002', 'ELDER_CARE', 'open', 'medium',
 'Smestaj u ustanovu - starija osoba bez porodice',
 'Osoba (78g) bez porodicnog staranja, potreban smestaj u Gerontoloski centar.',
 'a0000000-0000-0000-0000-000000000100', 'b0000000-0000-0000-0000-000000000004',
 NOW() - INTERVAL '7 days', NOW());

-- Slucaj 3: Nasilje u porodici
INSERT INTO cases.cases (id, case_number, type, status, priority, title, description, owning_agency_id, lead_worker_id, created_at, updated_at) VALUES
('c0000000-0000-0000-0000-000000000003', 'KI-2024-003', 'DOMESTIC_VIOLENCE', 'in_progress', 'urgent',
 'Zastita od nasilja u porodici',
 'Hitna intervencija - zastita zrtve nasilja. Koordinacija sa PU i Sigurnom kucom.',
 'a0000000-0000-0000-0000-000000000100', 'b0000000-0000-0000-0000-000000000003',
 NOW() - INTERVAL '3 days', NOW());

-- Slucaj 4: Osoba sa invaliditetom
INSERT INTO cases.cases (id, case_number, type, status, priority, title, description, owning_agency_id, lead_worker_id, created_at, updated_at) VALUES
('c0000000-0000-0000-0000-000000000004', 'KI-2024-004', 'DISABILITY_SUPPORT', 'open', 'medium',
 'Procena potreba za licnim pratiocem',
 'Zahtev za procenu potreba za licnim pratiocem za osobu sa cerebralnom paralizom.',
 'a0000000-0000-0000-0000-000000000100', 'b0000000-0000-0000-0000-000000000005',
 NOW() - INTERVAL '5 days', NOW());

-- Slucaj 5: Materijalna pomoc
INSERT INTO cases.cases (id, case_number, type, status, priority, title, description, owning_agency_id, lead_worker_id, closed_at, created_at, updated_at) VALUES
('c0000000-0000-0000-0000-000000000005', 'KI-2024-005', 'SOCIAL_ASSISTANCE', 'closed', 'low',
 'Jednokratna novcana pomoc',
 'Odobren zahtev za jednokratnu novcanu pomoc porodici u stanju socijalne potrebe.',
 'a0000000-0000-0000-0000-000000000100', 'b0000000-0000-0000-0000-000000000004',
 NOW() - INTERVAL '7 days',
 NOW() - INTERVAL '21 days', NOW());

-- Slucaj 6: Dete iz Sente - medjuopstinska koordinacija
INSERT INTO cases.cases (id, case_number, type, status, priority, title, description, owning_agency_id, lead_worker_id, created_at, updated_at) VALUES
('c0000000-0000-0000-0000-000000000006', 'SE-2024-001', 'CHILD_PROTECTION', 'in_progress', 'high',
 'Transfer slucaja - preseljenje porodice',
 'Porodica se preselila iz Sente u Kikindu. Potreban transfer slucaja i dokumentacije.',
 'a0000000-0000-0000-0000-000000000110', 'b0000000-0000-0000-0000-000000000003',
 NOW() - INTERVAL '10 days', NOW());

-- ============================================
-- CASE EVENTS (timeline)
-- ============================================

INSERT INTO cases.case_events (id, case_id, type, actor_id, actor_agency_id, description, timestamp) VALUES
-- Slucaj 1 events
('ce000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'CREATED', 'b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000100', 'Slucaj kreiran na osnovu prijave skole', NOW() - INTERVAL '14 days'),
('ce000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000001', 'OPENED', 'b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000100', 'Slucaj otvoren, dodeljen radnik', NOW() - INTERVAL '14 days'),
('ce000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001', 'SHARED', 'b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000100', 'Slucaj podeljen sa DZ Kikinda', NOW() - INTERVAL '12 days'),
('ce000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000001', 'DOCUMENT_ADDED', 'b0000000-0000-0000-0000-000000000011', 'a0000000-0000-0000-0000-000000000102', 'Dodat medicinski nalaz', NOW() - INTERVAL '11 days'),
('ce000000-0000-0000-0000-000000000005', 'c0000000-0000-0000-0000-000000000001', 'STATUS_CHANGED', 'b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000100', 'Status promenjen u IN_PROGRESS', NOW() - INTERVAL '10 days'),

-- Slucaj 3 events
('ce000000-0000-0000-0000-000000000010', 'c0000000-0000-0000-0000-000000000003', 'CREATED', 'b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000100', 'Hitna prijava nasilja', NOW() - INTERVAL '3 days'),
('ce000000-0000-0000-0000-000000000011', 'c0000000-0000-0000-0000-000000000003', 'OPENED', 'b0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000100', 'HITAN slucaj - odmah otvoren', NOW() - INTERVAL '3 days'),
('ce000000-0000-0000-0000-000000000012', 'c0000000-0000-0000-0000-000000000003', 'SHARED', 'b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000100', 'Podeljen sa PU Kikinda', NOW() - INTERVAL '3 days'),
('ce000000-0000-0000-0000-000000000013', 'c0000000-0000-0000-0000-000000000003', 'DOCUMENT_ADDED', 'b0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000100', 'Dodato resenje o hitnom zbrinjavanju', NOW() - INTERVAL '2 days');

-- ============================================
-- DOKUMENTI
-- ============================================

INSERT INTO documents.documents (id, document_number, type, status, title, description, owner_agency_id, created_by, case_id, current_version, created_at, updated_at) VALUES
-- Dokumenti za slucaj 1
('d0000000-0000-0000-0000-000000000001', 'DOC-KI-2024-0001', 'CASE_INTAKE', 'active', 'Zapisnik o prijemu slucaja', 'Prijava skole o sumnji na zanemarivanje', 'a0000000-0000-0000-0000-000000000100', 'b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001', 1, NOW() - INTERVAL '14 days', NOW()),
('d0000000-0000-0000-0000-000000000002', 'DOC-KI-2024-0002', 'MEDICAL_REPORT', 'active', 'Medicinski nalaz', 'Nalaz lekara o stanju deteta', 'a0000000-0000-0000-0000-000000000102', 'b0000000-0000-0000-0000-000000000011', 'c0000000-0000-0000-0000-000000000001', 1, NOW() - INTERVAL '11 days', NOW()),
('d0000000-0000-0000-0000-000000000003', 'DOC-KI-2024-0003', 'ASSESSMENT', 'active', 'Procena porodicne situacije', 'Izvestaj o poseti porodici', 'a0000000-0000-0000-0000-000000000100', 'b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000001', 2, NOW() - INTERVAL '9 days', NOW()),

-- Dokumenti za slucaj 2
('d0000000-0000-0000-0000-000000000004', 'DOC-KI-2024-0004', 'CASE_INTAKE', 'active', 'Zahtev za smestaj', 'Zahtev za smestaj starije osobe', 'a0000000-0000-0000-0000-000000000100', 'b0000000-0000-0000-0000-000000000004', 'c0000000-0000-0000-0000-000000000002', 1, NOW() - INTERVAL '7 days', NOW()),
('d0000000-0000-0000-0000-000000000005', 'DOC-KI-2024-0005', 'MEDICAL_REPORT', 'active', 'Medicinska dokumentacija', 'Dokumentacija o zdravstvenom stanju', 'a0000000-0000-0000-0000-000000000102', 'b0000000-0000-0000-0000-000000000012', 'c0000000-0000-0000-0000-000000000002', 1, NOW() - INTERVAL '5 days', NOW()),

-- Dokumenti za slucaj 3
('d0000000-0000-0000-0000-000000000006', 'DOC-KI-2024-0006', 'CASE_INTAKE', 'active', 'Prijava nasilja', 'Zapisnik o prijavi nasilja u porodici', 'a0000000-0000-0000-0000-000000000100', 'b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000003', 1, NOW() - INTERVAL '3 days', NOW()),
('d0000000-0000-0000-0000-000000000007', 'DOC-KI-2024-0007', 'DECISION', 'active', 'Resenje o hitnom zbrinjavanju', 'Doneto resenje o hitnom zbrinjavanju zrtve i maloletnog deteta u Sigurnu kucu.', 'a0000000-0000-0000-0000-000000000100', 'b0000000-0000-0000-0000-000000000003', 'c0000000-0000-0000-0000-000000000003', 1, NOW() - INTERVAL '3 days', NOW());

-- ============================================
-- AUDIT LOG
-- ============================================
-- NOTE: Audit entries are NOT pre-seeded with fake data.
-- They are created by the application when actual actions occur.
-- This ensures proper hash calculation and chain integrity.
--
-- To generate audit entries:
-- 1. Run a simulation from the Demo UI
-- 2. Each action in the simulation creates a proper audit entry
-- 3. The hash chain is built correctly by the application

-- Report counts
DO $$
BEGIN
  RAISE NOTICE 'Seed data loaded:';
  RAISE NOTICE '  - Agencies: %', (SELECT COUNT(*) FROM identity.agencies);
  RAISE NOTICE '  - Workers: %', (SELECT COUNT(*) FROM identity.workers);
  RAISE NOTICE '  - Cases: %', (SELECT COUNT(*) FROM cases.cases);
  RAISE NOTICE '  - Documents: %', (SELECT COUNT(*) FROM documents.documents);
  RAISE NOTICE '  - Case Events: %', (SELECT COUNT(*) FROM cases.case_events);
  RAISE NOTICE '  - Audit Log: %', (SELECT COUNT(*) FROM audit.entries);
END $$;
