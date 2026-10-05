-- Data contoh untuk warehouse_db (aman dijalankan sekali; ulangi = "Duplicate entry" untuk SKU/kode yang sama)
USE warehouse_db;

-- Kategori & lokasi
INSERT IGNORE INTO categories (name) VALUES ('Gypsum'), ('Rangka Plafon'), ('Aksesoris'), ('Finishing');
INSERT IGNORE INTO locations (name, description) VALUES
 ('Rak A1', 'Gypsum board'),
 ('Rak A2', 'Aksesoris & sekrup'),
 ('Rak B1', 'Hollow & rangka'),
 ('Gudang Belakang', 'Barang berat / sak');

-- Barang toko
INSERT INTO items (sku, name, category, location, quantity, min_threshold, unit, price, description) VALUES
 ('GYP-9MM',  'Gypsum Board 9mm 120x240',        'Gypsum',        'Rak A1',          120, 30, 'lembar', 62000,  'Gypsum standar plafon'),
 ('GYP-12MM', 'Gypsum Board 12mm 120x240',       'Gypsum',        'Rak A1',           18, 25, 'lembar', 85000,  'Untuk dinding partisi'),
 ('HOL-4X4',  'Hollow 4x4 galvanis 0.35mm',      'Rangka Plafon', 'Rak B1',          200, 50, 'batang', 21000,  ''),
 ('HOL-2X4',  'Hollow 2x4 galvanis 0.35mm',      'Rangka Plafon', 'Rak B1',          150, 50, 'batang', 17500,  ''),
 ('CRN-PVC',  'Cornis PVC 3m',                   'Aksesoris',     'Rak A2',           85, 20, 'batang', 28000,  ''),
 ('SKR-GYP',  'Sekrup Gypsum 1" (box 1000 pcs)', 'Aksesoris',     'Rak A2',            9, 10, 'box',    95000,  ''),
 ('PLM-25KG', 'Plamir Dinding 25kg',             'Finishing',     'Gudang Belakang',  34, 10, 'sak',    118000, ''),
 ('TAPE-JNT', 'Kertas Jointing Tape 90m',        'Finishing',     'Rak A2',           40, 15, 'roll',   12000,  '');

-- Riwayat mutasi stok (masuk - keluar = stok sekarang)
INSERT INTO stock_movements (item_id, type, quantity, reference, created_at)
SELECT id, 'IN', 150, 'Stok awal / pembelian supplier', '2026-09-01 09:00:00' FROM items WHERE sku='GYP-9MM' UNION ALL
SELECT id, 'OUT', 30, 'Penjualan proyek Way Halim',     '2026-09-18 14:30:00' FROM items WHERE sku='GYP-9MM' UNION ALL
SELECT id, 'IN', 60,  'Stok awal / pembelian supplier', '2026-09-01 09:00:00' FROM items WHERE sku='GYP-12MM' UNION ALL
SELECT id, 'OUT', 42, 'Penjualan partisi kantor',       '2026-09-25 10:15:00' FROM items WHERE sku='GYP-12MM' UNION ALL
SELECT id, 'IN', 250, 'Stok awal / pembelian supplier', '2026-09-02 09:00:00' FROM items WHERE sku='HOL-4X4' UNION ALL
SELECT id, 'OUT', 50, 'Penjualan proyek Kedaton',       '2026-09-22 11:00:00' FROM items WHERE sku='HOL-4X4' UNION ALL
SELECT id, 'IN', 200, 'Stok awal / pembelian supplier', '2026-09-02 09:00:00' FROM items WHERE sku='HOL-2X4' UNION ALL
SELECT id, 'OUT', 50, 'Penjualan eceran',               '2026-09-28 16:20:00' FROM items WHERE sku='HOL-2X4' UNION ALL
SELECT id, 'IN', 100, 'Stok awal / pembelian supplier', '2026-09-03 09:00:00' FROM items WHERE sku='CRN-PVC' UNION ALL
SELECT id, 'OUT', 15, 'Penjualan eceran',               '2026-09-29 13:45:00' FROM items WHERE sku='CRN-PVC' UNION ALL
SELECT id, 'IN', 20,  'Stok awal / pembelian supplier', '2026-09-03 09:00:00' FROM items WHERE sku='SKR-GYP' UNION ALL
SELECT id, 'OUT', 11, 'Pemakaian proyek',               '2026-10-02 15:10:00' FROM items WHERE sku='SKR-GYP' UNION ALL
SELECT id, 'IN', 40,  'Stok awal / pembelian supplier', '2026-09-05 09:00:00' FROM items WHERE sku='PLM-25KG' UNION ALL
SELECT id, 'OUT', 6,  'Penjualan proyek Teluk Betung',  '2026-10-03 10:40:00' FROM items WHERE sku='PLM-25KG' UNION ALL
SELECT id, 'IN', 50,  'Stok awal / pembelian supplier', '2026-09-05 09:00:00' FROM items WHERE sku='TAPE-JNT' UNION ALL
SELECT id, 'OUT', 10, 'Penjualan eceran',               '2026-10-04 12:05:00' FROM items WHERE sku='TAPE-JNT';

-- Sewa kapolding
INSERT INTO item_loans (loan_code, item_id, item_name, quantity, elbow_count, shock_count, borrower_name, borrower_phone,
  project_location, id_card_given, loan_date, due_date, return_date, status, rental_fee, owner_cost, notes) VALUES
 ('KPD-0001', NULL, 'Kapolding / Steger Set 170cm', 10, 20, 20, 'Pak Hendra',     '081234567801', 'Proyek Ruko Way Halim',        TRUE,  '2026-09-10', '2026-10-10', NULL,         'ACTIVE',   1500000, 600000, ''),
 ('KPD-0002', NULL, 'Kapolding / Steger Set 170cm',  6, 12, 12, 'Bu Sari',        '081399882201', 'Renovasi Rumah Kedaton',       FALSE, '2026-09-22', '2026-10-22', NULL,         'ACTIVE',    900000, 360000, ''),
 ('KPD-0003', NULL, 'Kapolding / Steger Set 170cm', 15, 30, 30, 'CV Mitra Bangun','082155443322', 'Gedung Kantor Teluk Betung',   TRUE,  '2026-08-15', '2026-09-15', '2026-09-14', 'RETURNED', 2250000, 900000, 'Kembali lengkap'),
 ('KPD-0004', NULL, 'Kapolding / Steger Set 170cm',  8, 16, 16, 'Pak Joko',       '085711229090', 'Pagar Gudang Tanjung Karang',  TRUE,  '2026-10-01', '2026-10-31', NULL,         'ACTIVE',   1200000, 480000, '');

-- Status pembayaran per bulan
INSERT INTO loan_monthly_settlements (loan_id, period, rental_fee, owner_cost, is_paid, is_paid_to_owner)
SELECT id, '2026-09', 1500000, 600000, TRUE,  TRUE  FROM item_loans WHERE loan_code='KPD-0001' UNION ALL
SELECT id, '2026-10', 1500000, 600000, FALSE, FALSE FROM item_loans WHERE loan_code='KPD-0001' UNION ALL
SELECT id, '2026-09',  900000, 360000, TRUE,  FALSE FROM item_loans WHERE loan_code='KPD-0002' UNION ALL
SELECT id, '2026-10',  900000, 360000, TRUE,  FALSE FROM item_loans WHERE loan_code='KPD-0002' UNION ALL
SELECT id, '2026-08', 2250000, 900000, TRUE,  TRUE  FROM item_loans WHERE loan_code='KPD-0003' UNION ALL
SELECT id, '2026-09', 2250000, 900000, TRUE,  TRUE  FROM item_loans WHERE loan_code='KPD-0003' UNION ALL
SELECT id, '2026-10', 1200000, 480000, FALSE, FALSE FROM item_loans WHERE loan_code='KPD-0004';
