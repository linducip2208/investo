package scraper

import (
	"fmt"
	"investo/internal/model"
	"strings"
	"time"
)

type IDXScraper struct{}

var seededStocks = []model.Stock{
	{Code: "BBCA", Name: "Bank Central Asia Tbk", SectorID: 8},
	{Code: "BBRI", Name: "Bank Rakyat Indonesia Tbk", SectorID: 8},
	{Code: "BMRI", Name: "Bank Mandiri Tbk", SectorID: 8},
	{Code: "BBNI", Name: "Bank Negara Indonesia Tbk", SectorID: 8},
	{Code: "BBTN", Name: "Bank Tabungan Negara Tbk", SectorID: 8},
	{Code: "BJBR", Name: "Bank Pembangunan Daerah Jawa Barat Tbk", SectorID: 8},
	{Code: "BJTM", Name: "Bank Pembangunan Daerah Jawa Timur Tbk", SectorID: 8},
	{Code: "BFIN", Name: "BFI Finance Indonesia Tbk", SectorID: 8},
	{Code: "PNBN", Name: "Bank Pan Indonesia Tbk", SectorID: 8},
	{Code: "NISP", Name: "Bank OCBC NISP Tbk", SectorID: 8},
	{Code: "BTPN", Name: "Bank BTPN Tbk", SectorID: 8},
	{Code: "BMAS", Name: "Bank Maspion Indonesia Tbk", SectorID: 8},
	{Code: "AMAR", Name: "Bank Amar Indonesia Tbk", SectorID: 8},
	{Code: "AGRO", Name: "Bank Raya Indonesia Tbk", SectorID: 8},
	{Code: "BNBA", Name: "Bank Bumi Arta Tbk", SectorID: 8},
	{Code: "BKSW", Name: "Bank QNB Indonesia Tbk", SectorID: 8},
	{Code: "BNGA", Name: "Bank CIMB Niaga Tbk", SectorID: 8},
	{Code: "BNII", Name: "Bank Maybank Indonesia Tbk", SectorID: 8},
	{Code: "BSWD", Name: "Bank of India Indonesia Tbk", SectorID: 8},
	{Code: "MAYA", Name: "Bank Mayapada Internasional Tbk", SectorID: 8},
	{Code: "MCOR", Name: "Bank China Construction Bank Indonesia Tbk", SectorID: 8},
	{Code: "MEGA", Name: "Bank Mega Tbk", SectorID: 8},
	{Code: "DNAR", Name: "Bank Dinar Indonesia Tbk", SectorID: 8},
	{Code: "BBKP", Name: "Bank KB Bukopin Tbk", SectorID: 8},
	{Code: "BDMN", Name: "Bank Danamon Indonesia Tbk", SectorID: 8},
	{Code: "INPC", Name: "Bank Artha Graha Internasional Tbk", SectorID: 8},
	{Code: "BVIC", Name: "Bank Victoria International Tbk", SectorID: 8},
	{Code: "SDRA", Name: "Bank Woori Saudara Indonesia 1906 Tbk", SectorID: 8},
	{Code: "BABP", Name: "Bank MNC Internasional Tbk", SectorID: 8},
	{Code: "BBMD", Name: "Bank Mestika Dharma Tbk", SectorID: 8},
	{Code: "BBNP", Name: "Bank Nusantara Parahyangan Tbk", SectorID: 8},
	{Code: "BCIC", Name: "Bank JTrust Indonesia Tbk", SectorID: 8},
	{Code: "BGTB", Name: "Bank Ganesha Tbk", SectorID: 8},
	{Code: "BHAT", Name: "Bank Bhakti Tbk", SectorID: 8},
	{Code: "BMTR", Name: "Bank Mutiara Tbk", SectorID: 8},
	{Code: "BNLI", Name: "Bank Permata Tbk", SectorID: 8},
	{Code: "BPFI", Name: "Bank Ganesha Tbk", SectorID: 8},
	{Code: "BSIM", Name: "Bank Sinarmas Tbk", SectorID: 8},
	{Code: "GTBO", Name: "Garda Tujuh Buana Tbk", SectorID: 8},
	{Code: "HDFA", Name: "Radana Bhaskara Finance Tbk", SectorID: 8},
	{Code: "IBST", Name: "Inti Bangun Sejahtera Tbk", SectorID: 8},
	{Code: "LCKM", Name: "LCK Global Kedaton Tbk", SectorID: 8},
	{Code: "MFIN", Name: "Mandala Multifinance Tbk", SectorID: 8},
	{Code: "ADMF", Name: "Adira Dinamika Multi Finance Tbk", SectorID: 8},
	{Code: "BTPN", Name: "Bank BTPN Syariah Tbk", SectorID: 8},
	{Code: "IMJS", Name: "Indomobil Multi Jasa Tbk", SectorID: 8},
	{Code: "BPFI", Name: "Batavia Prosperindo Finance Tbk", SectorID: 8},
	{Code: "LPGI", Name: "Lippo General Insurance Tbk", SectorID: 8},
	{Code: "ASBI", Name: "Asuransi Bintang Tbk", SectorID: 8},
	{Code: "ASDM", Name: "Asuransi Dayin Mitra Tbk", SectorID: 8},
	{Code: "ASMI", Name: "Asuransi Maximus Graha Persada Tbk", SectorID: 8},
	{Code: "PNLF", Name: "Panin Financial Tbk", SectorID: 8},
	{Code: "PEGE", Name: "Panca Global Kapital Tbk", SectorID: 8},
	{Code: "RELI", Name: "Reliance Sekuritas Indonesia Tbk", SectorID: 8},
	{Code: "TRUS", Name: "Trust Finance Indonesia Tbk", SectorID: 8},
	{Code: "VRNA", Name: "Verena Multi Finance Tbk", SectorID: 8},

	{Code: "TLKM", Name: "Telkom Indonesia Tbk", SectorID: 11},
	{Code: "ASII", Name: "Astra International Tbk", SectorID: 4},

	{Code: "UNVR", Name: "Unilever Indonesia Tbk", SectorID: 1},
	{Code: "ICBP", Name: "Indofood CBP Sukses Makmur Tbk", SectorID: 1},
	{Code: "INDF", Name: "Indofood Sukses Makmur Tbk", SectorID: 1},
	{Code: "MYOR", Name: "Mayora Indah Tbk", SectorID: 1},
	{Code: "HMSP", Name: "HM Sampoerna Tbk", SectorID: 1},
	{Code: "GGRM", Name: "Gudang Garam Tbk", SectorID: 1},
	{Code: "AMRT", Name: "Sumber Alfaria Trijaya Tbk", SectorID: 1},
	{Code: "CPIN", Name: "Charoen Pokphand Indonesia Tbk", SectorID: 1},
	{Code: "JPFA", Name: "Japfa Comfeed Indonesia Tbk", SectorID: 1},
	{Code: "ULTJ", Name: "Ultrajaya Milk Industry Tbk", SectorID: 1},
	{Code: "ROTI", Name: "Nippon Indosari Corpindo Tbk", SectorID: 1},
	{Code: "CLEO", Name: "Sariguna Primatirta Tbk", SectorID: 1},
	{Code: "COCO", Name: "Wahana Interfood Nusantara Tbk", SectorID: 1},
	{Code: "CAMP", Name: "Campina Ice Cream Industry Tbk", SectorID: 1},
	{Code: "CEKA", Name: "Wilmar Cahaya Indonesia Tbk", SectorID: 1},
	{Code: "SKLT", Name: "Sekar Laut Tbk", SectorID: 1},
	{Code: "STTP", Name: "Siantar Top Tbk", SectorID: 1},
	{Code: "ALTO", Name: "Tri Banyan Tirta Tbk", SectorID: 1},
	{Code: "AISA", Name: "FKS Food Sejahtera Tbk", SectorID: 1},
	{Code: "BTEK", Name: "Bumi Teknokultura Unggul Tbk", SectorID: 1},
	{Code: "BUDI", Name: "Budi Starch & Sweetener Tbk", SectorID: 1},
	{Code: "DLTA", Name: "Delta Djakarta Tbk", SectorID: 1},
	{Code: "HOKI", Name: "Buyung Poetra Sembada Tbk", SectorID: 1},
	{Code: "KEJU", Name: "Mulia Boga Raya Tbk", SectorID: 1},
	{Code: "MLBI", Name: "Multi Bintang Indonesia Tbk", SectorID: 1},
	{Code: "PSDN", Name: "Prasidha Aneka Niaga Tbk", SectorID: 1},
	{Code: "SKBM", Name: "Sekar Bumi Tbk", SectorID: 1},
	{Code: "WINE", Name: "Hatten Bali Tbk", SectorID: 1},
	{Code: "ADES", Name: "Akasha Wira International Tbk", SectorID: 1},
	{Code: "VICI", Name: "Victoria Care Indonesia Tbk", SectorID: 1},
	{Code: "KINO", Name: "Kino Indonesia Tbk", SectorID: 1},
	{Code: "GOOD", Name: "Garudafood Putra Putri Jaya Tbk", SectorID: 1},
	{Code: "TGKA", Name: "Tigaraksa Satria Tbk", SectorID: 1},
	{Code: "FOOD", Name: "Sentra Food Indonesia Tbk", SectorID: 1},
	{Code: "PCAR", Name: "Prima Cakrawala Abadi Tbk", SectorID: 1},
	{Code: "PSGO", Name: "Palma Serasih Tbk", SectorID: 1},
	{Code: "WAPO", Name: "Wahana Ottomitra Multiartha Tbk", SectorID: 1},
	{Code: "SIDO", Name: "Industri Jamu dan Farmasi Sido Muncul Tbk", SectorID: 6},
	{Code: "KLBF", Name: "Kalbe Farma Tbk", SectorID: 6},
	{Code: "KAEF", Name: "Kimia Farma Tbk", SectorID: 6},
	{Code: "MERK", Name: "Merck Tbk", SectorID: 6},
	{Code: "DVLA", Name: "Darya-Varia Laboratoria Tbk", SectorID: 6},
	{Code: "PEHA", Name: "Phapros Tbk", SectorID: 6},
	{Code: "PYFA", Name: "Pyridam Farma Tbk", SectorID: 6},
	{Code: "TSPC", Name: "Tempo Scan Pacific Tbk", SectorID: 6},
	{Code: "MIKA", Name: "Mitra Keluarga Karyasehat Tbk", SectorID: 6},
	{Code: "HEAL", Name: "Medikaloka Hermina Tbk", SectorID: 6},
	{Code: "SAME", Name: "Sarana Meditama Metropolitan Tbk", SectorID: 6},
	{Code: "SILO", Name: "Siloam International Hospitals Tbk", SectorID: 6},
	{Code: "IRRA", Name: "Itama Ranoraya Tbk", SectorID: 6},
	{Code: "PRDA", Name: "Prodia Widyahusada Tbk", SectorID: 6},
	{Code: "SOBI", Name: "Sorini Agro Asia Corporindo Tbk", SectorID: 6},
	{Code: "CARE", Name: "Metro Healthcare Indonesia Tbk", SectorID: 6},

	{Code: "ADRO", Name: "Adaro Energy Indonesia Tbk", SectorID: 3},
	{Code: "PTBA", Name: "Bukit Asam Tbk", SectorID: 3},
	{Code: "ITMG", Name: "Indo Tambangraya Megah Tbk", SectorID: 3},
	{Code: "ANTM", Name: "Aneka Tambang Tbk", SectorID: 3},
	{Code: "INCO", Name: "Vale Indonesia Tbk", SectorID: 3},
	{Code: "MDKA", Name: "Merdeka Copper Gold Tbk", SectorID: 3},
	{Code: "PGAS", Name: "Perusahaan Gas Negara Tbk", SectorID: 3},
	{Code: "BUMI", Name: "Bumi Resources Tbk", SectorID: 3},
	{Code: "MEDC", Name: "Medco Energi Internasional Tbk", SectorID: 3},
	{Code: "HRUM", Name: "Harum Energy Tbk", SectorID: 3},
	{Code: "PTRO", Name: "Petrosea Tbk", SectorID: 3},
	{Code: "DOID", Name: "Delta Dunia Makmur Tbk", SectorID: 3},
	{Code: "BSSR", Name: "Baramulti Suksessarana Tbk", SectorID: 3},
	{Code: "INDY", Name: "Indika Energy Tbk", SectorID: 3},
	{Code: "ELSA", Name: "Elnusa Tbk", SectorID: 3},
	{Code: "MBSS", Name: "Mitrabahtera Segara Sejati Tbk", SectorID: 3},
	{Code: "ENRG", Name: "Energi Mega Persada Tbk", SectorID: 3},
	{Code: "TINS", Name: "Timah Tbk", SectorID: 3},
	{Code: "RMKE", Name: "RMK Energy Tbk", SectorID: 3},
	{Code: "TOBA", Name: "TBS Energi Utama Tbk", SectorID: 3},
	{Code: "BYAN", Name: "Bayan Resources Tbk", SectorID: 3},
	{Code: "ARII", Name: "Atlas Resources Tbk", SectorID: 3},
	{Code: "ATPK", Name: "Bara Jasa Internasional Tbk", SectorID: 3},
	{Code: "BOSS", Name: "Borneo Olah Sarana Sukses Tbk", SectorID: 3},
	{Code: "CITA", Name: "Cita Mineral Investindo Tbk", SectorID: 3},
	{Code: "CTBN", Name: "Citra Tubindo Tbk", SectorID: 3},
	{Code: "DEWA", Name: "Darma Henwa Tbk", SectorID: 3},
	{Code: "DKFT", Name: "Central Omega Resources Tbk", SectorID: 3},
	{Code: "DSSA", Name: "Dian Swastatika Sentosa Tbk", SectorID: 3},
	{Code: "ESSA", Name: "Surya Esa Perkasa Tbk", SectorID: 3},
	{Code: "FIRE", Name: "Alfa Energi Investama Tbk", SectorID: 3},
	{Code: "GEMS", Name: "Golden Energy Mines Tbk", SectorID: 3},
	{Code: "IFSH", Name: "Ifishdeco Tbk", SectorID: 3},
	{Code: "KKGI", Name: "Resource Alam Indonesia Tbk", SectorID: 3},
	{Code: "MITI", Name: "Mitra Investindo Tbk", SectorID: 3},
	{Code: "MTFN", Name: "Capitalinc Investment Tbk", SectorID: 3},
	{Code: "MYOH", Name: "Samindo Resources Tbk", SectorID: 3},
	{Code: "PKPK", Name: "Perdana Karya Perkasa Tbk", SectorID: 3},
	{Code: "RUIS", Name: "Radiant Utama Interinsco Tbk", SectorID: 3},
	{Code: "SMMT", Name: "Golden Eagle Energy Tbk", SectorID: 3},
	{Code: "SOCI", Name: "Soechi Lines Tbk", SectorID: 3},
	{Code: "SUGI", Name: "Sugih Energy Tbk", SectorID: 3},
	{Code: "TPMA", Name: "Trans Power Marine Tbk", SectorID: 3},
	{Code: "ZINC", Name: "Kapuas Prima Coal Tbk", SectorID: 3},
	{Code: "AKRA", Name: "AKR Corporindo Tbk", SectorID: 3},
	{Code: "RAJA", Name: "Rukun Raharja Tbk", SectorID: 3},
	{Code: "POWR", Name: "Cikarang Listrindo Tbk", SectorID: 3},
	{Code: "SMRU", Name: "SMR Utama Tbk", SectorID: 3},
	{Code: "BIPI", Name: "Benakat Integra Tbk", SectorID: 3},

	{Code: "UNTR", Name: "United Tractors Tbk", SectorID: 4},
	{Code: "GJTL", Name: "Gajah Tunggal Tbk", SectorID: 4},
	{Code: "KBLI", Name: "KMI Wire and Cable Tbk", SectorID: 4},
	{Code: "DRMA", Name: "Dharma Polimetal Tbk", SectorID: 4},
	{Code: "AUTO", Name: "Astra Otoparts Tbk", SectorID: 4},
	{Code: "IMAS", Name: "Indomobil Sukses Internasional Tbk", SectorID: 4},
	{Code: "MASA", Name: "Multistrada Arah Sarana Tbk", SectorID: 4},
	{Code: "PRAS", Name: "Prima Alloy Steel Universal Tbk", SectorID: 4},
	{Code: "SMSM", Name: "Selamat Sempurna Tbk", SectorID: 4},
	{Code: "BRAM", Name: "Indo Kordsa Tbk", SectorID: 4},
	{Code: "LPIN", Name: "Multi Prima Sejahtera Tbk", SectorID: 4},
	{Code: "NIPS", Name: "Nipress Tbk", SectorID: 4},
	{Code: "PIPA", Name: "Indorama Synthetics Tbk", SectorID: 4},
	{Code: "TURI", Name: "Tunas Ridean Tbk", SectorID: 4},

	{Code: "TBLA", Name: "Tunas Baru Lampung Tbk", SectorID: 5},
	{Code: "SIMP", Name: "Salim Ivomas Pratama Tbk", SectorID: 5},
	{Code: "MAIN", Name: "Malindo Feedmill Tbk", SectorID: 5},
	{Code: "LSIP", Name: "PP London Sumatra Indonesia Tbk", SectorID: 5},
	{Code: "SGRO", Name: "Sampoerna Agro Tbk", SectorID: 5},
	{Code: "AALI", Name: "Astra Agro Lestari Tbk", SectorID: 5},
	{Code: "SMAR", Name: "Smart Tbk", SectorID: 5},
	{Code: "SSMS", Name: "Sawit Sumbermas Sarana Tbk", SectorID: 5},
	{Code: "DSNG", Name: "Dharma Satya Nusantara Tbk", SectorID: 5},
	{Code: "JAWA", Name: "Jaya Agra Wattie Tbk", SectorID: 5},
	{Code: "MGRO", Name: "Mahkota Group Tbk", SectorID: 5},
	{Code: "ANJT", Name: "Austindo Nusantara Jaya Tbk", SectorID: 5},
	{Code: "BWPT", Name: "Eagle High Plantations Tbk", SectorID: 5},
	{Code: "GZCO", Name: "Gozco Plantations Tbk", SectorID: 5},
	{Code: "MAGP", Name: "Multi Agro Gemilang Plantation Tbk", SectorID: 5},
	{Code: "PALM", Name: "Provident Investasi Bersama Tbk", SectorID: 5},
	{Code: "RANC", Name: "Supra Boga Lestari Tbk", SectorID: 5},
	{Code: "UNSP", Name: "Bakrie Sumatera Plantations Tbk", SectorID: 5},

	{Code: "SMGR", Name: "Semen Indonesia Tbk", SectorID: 2},
	{Code: "INTP", Name: "Indocement Tunggal Prakarsa Tbk", SectorID: 2},
	{Code: "WIKA", Name: "Wijaya Karya Tbk", SectorID: 2},
	{Code: "PTPP", Name: "PP (Persero) Tbk", SectorID: 2},
	{Code: "ADHI", Name: "Adhi Karya Tbk", SectorID: 2},
	{Code: "WSKT", Name: "Waskita Karya Tbk", SectorID: 2},
	{Code: "BRPT", Name: "Barito Pacific Tbk", SectorID: 2},
	{Code: "TPIA", Name: "Chandra Asri Petrochemical Tbk", SectorID: 2},
	{Code: "INKP", Name: "Indah Kiat Pulp & Paper Tbk", SectorID: 2},
	{Code: "BDKR", Name: "Berdikari Pondasi Perkasa Tbk", SectorID: 2},
	{Code: "WTON", Name: "Wijaya Karya Beton Tbk", SectorID: 2},
	{Code: "KRAS", Name: "Krakatau Steel Tbk", SectorID: 2},
	{Code: "MARK", Name: "Mark Dynamics Indonesia Tbk", SectorID: 2},
	{Code: "IMPC", Name: "Impack Pratama Industri Tbk", SectorID: 2},
	{Code: "ISSP", Name: "Steel Pipe Industry of Indonesia Tbk", SectorID: 2},
	{Code: "TOTO", Name: "Surya Toto Indonesia Tbk", SectorID: 2},
	{Code: "WEGE", Name: "Wijaya Karya Bangunan Gedung Tbk", SectorID: 2},
	{Code: "AKPI", Name: "Argha Karya Prima Industry Tbk", SectorID: 2},
	{Code: "ALKA", Name: "Alakasa Industrindo Tbk", SectorID: 2},
	{Code: "ALMI", Name: "Alumindo Light Metal Industry Tbk", SectorID: 2},
	{Code: "APLI", Name: "Asiaplast Industries Tbk", SectorID: 2},
	{Code: "ARNA", Name: "Arwana Citramulia Tbk", SectorID: 2},
	{Code: "BAJA", Name: "Saranacentral Bajatama Tbk", SectorID: 2},
	{Code: "BTON", Name: "Betonjaya Manunggal Tbk", SectorID: 2},
	{Code: "CPRO", Name: "Central Proteina Prima Tbk", SectorID: 2},
	{Code: "CTBN", Name: "Citra Tubindo Tbk", SectorID: 2},
	{Code: "EKAD", Name: "Ekaputra Wijaya Lestari Tbk", SectorID: 2},
	{Code: "FPNI", Name: "Lotte Chemical Titan Tbk", SectorID: 2},
	{Code: "GDST", Name: "Gunawan Dianjaya Steel Tbk", SectorID: 2},
	{Code: "IGAR", Name: "Champion Pacific Indonesia Tbk", SectorID: 2},
	{Code: "INRU", Name: "Toba Pulp Lestari Tbk", SectorID: 2},
	{Code: "IPOL", Name: "Indopoly Swakarsa Industry Tbk", SectorID: 2},
	{Code: "JKSW", Name: "Jakarta Kyoei Steel Works Tbk", SectorID: 2},
	{Code: "KBRI", Name: "Kertas Basuki Rachmat Indonesia Tbk", SectorID: 2},
	{Code: "KDSI", Name: "Kedawung Setia Industrial Tbk", SectorID: 2},
	{Code: "LION", Name: "Lion Metal Works Tbk", SectorID: 2},
	{Code: "LMSH", Name: "Lionmesh Prima Tbk", SectorID: 2},
	{Code: "MLIA", Name: "Mulia Industrindo Tbk", SectorID: 2},
	{Code: "NIKL", Name: "Pelat Timah Nusantara Tbk", SectorID: 2},
	{Code: "PICO", Name: "Pelangi Indah Canindo Tbk", SectorID: 2},
	{Code: "SPMA", Name: "Suparma Tbk", SectorID: 2},
	{Code: "SRSN", Name: "Indo Acidatama Tbk", SectorID: 2},
	{Code: "TALF", Name: "Tunas Alfin Tbk", SectorID: 2},
	{Code: "TBMS", Name: "Tembaga Mulia Semanan Tbk", SectorID: 2},
	{Code: "TRST", Name: "Trias Sentosa Tbk", SectorID: 2},
	{Code: "UNIC", Name: "Unggul Indah Cahaya Tbk", SectorID: 2},
	{Code: "YPAS", Name: "Yanaprima Hastapersada Tbk", SectorID: 2},
	{Code: "SMCB", Name: "Solusi Bangun Indonesia Tbk", SectorID: 2},
	{Code: "CMNT", Name: "Cemindo Gemilang Tbk", SectorID: 2},
	{Code: "WSBP", Name: "Waskita Beton Precast Tbk", SectorID: 2},

	{Code: "BUKA", Name: "Bukalapak.com Tbk", SectorID: 7},
	{Code: "GOTO", Name: "GoTo Gojek Tokopedia Tbk", SectorID: 7},
	{Code: "EMTK", Name: "Elang Mahkota Teknologi Tbk", SectorID: 7},
	{Code: "MCAS", Name: "M Cash Integrasi Tbk", SectorID: 7},
	{Code: "MTDL", Name: "Metrodata Electronics Tbk", SectorID: 7},
	{Code: "DIVA", Name: "Distribusi Voucher Nusantara Tbk", SectorID: 7},
	{Code: "EDGE", Name: "Indointernet Tbk", SectorID: 7},
	{Code: "JAST", Name: "Jasnita Telekomindo Tbk", SectorID: 7},
	{Code: "KBLV", Name: "First Media Tbk", SectorID: 7},
	{Code: "LIVE", Name: "Matahari Lifestyle Indonesia Tbk", SectorID: 7},
	{Code: "MLPT", Name: "Multipolar Technology Tbk", SectorID: 7},
	{Code: "NASA", Name: "Ayana Land International Tbk", SectorID: 7},
	{Code: "NFCX", Name: "NFC Indonesia Tbk", SectorID: 7},
	{Code: "NICE", Name: "Adi Sarana Logistik Tbk", SectorID: 7},
	{Code: "SATU", Name: "Satu Visi Putra Tbk", SectorID: 7},
	{Code: "TFAS", Name: "Telefast Indonesia Tbk", SectorID: 7},
	{Code: "TMPO", Name: "Tempo Inti Media Tbk", SectorID: 7},
	{Code: "VISI", Name: "Indovisual Teknologi Tbk", SectorID: 7},
	{Code: "VIVA", Name: "Visi Media Asia Tbk", SectorID: 7},
	{Code: "WIFI", Name: "Solusi Sinergi Digital Tbk", SectorID: 7},
	{Code: "ZBRA", Name: "Zebra Nusantara Tbk", SectorID: 7},
	{Code: "TRIO", Name: "Trikomsel Oke Tbk", SectorID: 7},
	{Code: "GOLD", Name: "Visi Telekomunikasi Infrastruktur Tbk", SectorID: 7},
	{Code: "KIOS", Name: "Kioson Komersial Indonesia Tbk", SectorID: 7},
	{Code: "SKYB", Name: "Sky Energy Indonesia Tbk", SectorID: 7},

	{Code: "CTRA", Name: "Ciputra Development Tbk", SectorID: 9},
	{Code: "BSDE", Name: "Bumi Serpong Damai Tbk", SectorID: 9},
	{Code: "PWON", Name: "Pakuwon Jati Tbk", SectorID: 9},
	{Code: "SMRA", Name: "Summarecon Agung Tbk", SectorID: 9},
	{Code: "DILD", Name: "Intiland Development Tbk", SectorID: 9},
	{Code: "PPRO", Name: "PP Properti Tbk", SectorID: 9},
	{Code: "LPKR", Name: "Lippo Karawaci Tbk", SectorID: 9},
	{Code: "DMAS", Name: "Puradelta Lestari Tbk", SectorID: 9},
	{Code: "MTLA", Name: "Metropolitan Land Tbk", SectorID: 9},
	{Code: "RODA", Name: "Pikko Land Development Tbk", SectorID: 9},
	{Code: "APLN", Name: "Agung Podomoro Land Tbk", SectorID: 9},
	{Code: "ASRI", Name: "Alam Sutera Realty Tbk", SectorID: 9},
	{Code: "BAPA", Name: "Bekasi Asri Pemula Tbk", SectorID: 9},
	{Code: "BEST", Name: "Bekasi Fajar Industrial Estate Tbk", SectorID: 9},
	{Code: "BIPP", Name: "Bhuwanatala Indah Permai Tbk", SectorID: 9},
	{Code: "BKDP", Name: "Bukit Darmo Property Tbk", SectorID: 9},
	{Code: "BKSL", Name: "Sentul City Tbk", SectorID: 9},
	{Code: "COWL", Name: "Cowell Development Tbk", SectorID: 9},
	{Code: "DART", Name: "Duta Anggada Realty Tbk", SectorID: 9},
	{Code: "DUTI", Name: "Duta Pertiwi Tbk", SectorID: 9},
	{Code: "ELTY", Name: "Bakrieland Development Tbk", SectorID: 9},
	{Code: "FMII", Name: "Fortune Mate Indonesia Tbk", SectorID: 9},
	{Code: "GMTD", Name: "Gowa Makassar Tourism Development Tbk", SectorID: 9},
	{Code: "GPRA", Name: "Perdana Gapuraprima Tbk", SectorID: 9},
	{Code: "GRPM", Name: "Graha Prima Mentari Tbk", SectorID: 9},
	{Code: "GWSA", Name: "Greenwood Sejahtera Tbk", SectorID: 9},
	{Code: "JRPT", Name: "Jaya Real Property Tbk", SectorID: 9},
	{Code: "KIJA", Name: "Kawasan Industri Jababeka Tbk", SectorID: 9},
	{Code: "LAMI", Name: "Lamicitra Nusantara Tbk", SectorID: 9},
	{Code: "LCGP", Name: "Eureka Prima Jakarta Tbk", SectorID: 9},
	{Code: "LPCK", Name: "Lippo Cikarang Tbk", SectorID: 9},
	{Code: "MDLN", Name: "Modernland Realty Tbk", SectorID: 9},
	{Code: "MKPI", Name: "Metropolitan Kentjana Tbk", SectorID: 9},
	{Code: "MMLP", Name: "Mega Manunggal Property Tbk", SectorID: 9},
	{Code: "MTSM", Name: "Metro Realty Tbk", SectorID: 9},
	{Code: "NIRO", Name: "Nirvana Development Tbk", SectorID: 9},
	{Code: "OMRE", Name: "Indonesia Prima Property Tbk", SectorID: 9},
	{Code: "PAMG", Name: "Bima Sakti Pertiwi Tbk", SectorID: 9},
	{Code: "PLIN", Name: "Plaza Indonesia Realty Tbk", SectorID: 9},
	{Code: "PUDP", Name: "Pudjiadi Prestige Tbk", SectorID: 9},
	{Code: "RBMS", Name: "Ristia Bintang Mahkotasejati Tbk", SectorID: 9},
	{Code: "RDTX", Name: "Roda Vivatex Tbk", SectorID: 9},
	{Code: "SMDM", Name: "Suryamas Dutamakmur Tbk", SectorID: 9},
	{Code: "SMMA", Name: "Sinarmas Land Tbk", SectorID: 9},
	{Code: "TARA", Name: "Agung Semesta Sejahtera Tbk", SectorID: 9},
	{Code: "UANG", Name: "Bank Mayapada Internasional Tbk", SectorID: 9},
	{Code: "JRPT", Name: "Jaya Real Property Tbk", SectorID: 9},
	{Code: "KAII", Name: "Kairos Property Tbk", SectorID: 9},
	{Code: "URBN", Name: "Urban Jakarta Propertindo Tbk", SectorID: 9},
	{Code: "TOTL", Name: "Total Bangun Persada Tbk", SectorID: 9},

	{Code: "SRIL", Name: "Sri Rejeki Isman Tbk", SectorID: 10},
	{Code: "ERAA", Name: "Erajaya Swasembada Tbk", SectorID: 10},
	{Code: "ACES", Name: "Ace Hardware Indonesia Tbk", SectorID: 10},
	{Code: "MAPI", Name: "Mitra Adiperkasa Tbk", SectorID: 10},
	{Code: "LPPF", Name: "Matahari Department Store Tbk", SectorID: 10},
	{Code: "MNCN", Name: "Media Nusantara Citra Tbk", SectorID: 10},
	{Code: "SCMA", Name: "Surya Citra Media Tbk", SectorID: 10},
	{Code: "FILM", Name: "MD Pictures Tbk", SectorID: 10},
	{Code: "MAPA", Name: "MAP Aktif Adiperkasa Tbk", SectorID: 10},
	{Code: "HRTA", Name: "Hartadinata Abadi Tbk", SectorID: 10},
	{Code: "RALS", Name: "Ramayana Lestari Sentosa Tbk", SectorID: 10},
	{Code: "CSAP", Name: "Catur Sentosa Adiprana Tbk", SectorID: 10},
	{Code: "DAYA", Name: "Duta Intidaya Tbk", SectorID: 10},
	{Code: "ECII", Name: "Electronic City Indonesia Tbk", SectorID: 10},
	{Code: "FAST", Name: "Fast Food Indonesia Tbk", SectorID: 10},
	{Code: "GLOB", Name: "Globe Kita Terang Tbk", SectorID: 10},
	{Code: "HERO", Name: "Hero Supermarket Tbk", SectorID: 10},
	{Code: "HOME", Name: "Hotel Mandarine Regency Tbk", SectorID: 10},
	{Code: "JARR", Name: "Jaya Agra Wattie Tbk", SectorID: 10},
	{Code: "JGLE", Name: "Jaya Real Property Tbk", SectorID: 10},
	{Code: "KEEN", Name: "Kencana Energi Lestari Tbk", SectorID: 10},
	{Code: "KOIN", Name: "Kokoh Inti Arebama Tbk", SectorID: 10},
	{Code: "KPIG", Name: "MNC Land Tbk", SectorID: 10},
	{Code: "MIDI", Name: "Midi Utama Indonesia Tbk", SectorID: 10},
	{Code: "MKNT", Name: "Mitra Komunikasi Nusantara Tbk", SectorID: 10},
	{Code: "MPPA", Name: "Matahari Putra Prima Tbk", SectorID: 10},
	{Code: "PANR", Name: "Panorama Sentrawisata Tbk", SectorID: 10},
	{Code: "PJAA", Name: "Pembangunan Jaya Ancol Tbk", SectorID: 10},
	{Code: "PLAN", Name: "Planet Properindo Jaya Tbk", SectorID: 10},
	{Code: "PZZA", Name: "Sarimelati Kencana Tbk", SectorID: 10},
	{Code: "RIMO", Name: "Rimo International Lestari Tbk", SectorID: 10},
	{Code: "SHIP", Name: "Sillo Maritime Perdana Tbk", SectorID: 10},
	{Code: "SOHO", Name: "Soho Global Health Tbk", SectorID: 10},
	{Code: "SONA", Name: "Sona Topas Tourism Industry Tbk", SectorID: 10},
	{Code: "SQMI", Name: "Wilton Makmur Indonesia Tbk", SectorID: 10},
	{Code: "TCID", Name: "Mandom Indonesia Tbk", SectorID: 10},
	{Code: "TELE", Name: "Tiphone Mobile Indonesia Tbk", SectorID: 10},
	{Code: "TUGU", Name: "Asuransi Tugu Pratama Indonesia Tbk", SectorID: 10},
	{Code: "VOKS", Name: "Voksel Electric Tbk", SectorID: 10},
	{Code: "WICO", Name: "Wicaksana Overseas International Tbk", SectorID: 10},
	{Code: "WOOD", Name: "Integra Indocabinet Tbk", SectorID: 10},
	{Code: "KRAH", Name: "Grand Kartech Tbk", SectorID: 10},
	{Code: "MSKY", Name: "MNC Sky Vision Tbk", SectorID: 10},
	{Code: "BMTR", Name: "Global Mediacom Tbk", SectorID: 10},
	{Code: "CNKO", Name: "Bara Jaya Internasional Tbk", SectorID: 10},
	{Code: "DEFI", Name: "Danasupra Erapacific Tbk", SectorID: 10},
	{Code: "INPP", Name: "Indonesian Paradise Property Tbk", SectorID: 10},
	{Code: "KICI", Name: "Kedaung Indah Can Tbk", SectorID: 10},
	{Code: "MAMI", Name: "Mas Murni Indonesia Tbk", SectorID: 10},
	{Code: "PGLI", Name: "Pembangunan Graha Lestari Indah Tbk", SectorID: 10},
	{Code: "POLY", Name: "Asia Pacific Fibers Tbk", SectorID: 10},
	{Code: "IPOL", Name: "Indopoly Swakarsa Industry Tbk", SectorID: 10},

	{Code: "TOWR", Name: "Sarana Menara Nusantara Tbk", SectorID: 11},
	{Code: "EXCL", Name: "XL Axiata Tbk", SectorID: 11},
	{Code: "ISAT", Name: "Indosat Tbk", SectorID: 11},
	{Code: "MTEL", Name: "Dayamitra Telekomunikasi Tbk", SectorID: 11},
	{Code: "JSMR", Name: "Jasa Marga Tbk", SectorID: 11},
	{Code: "TBIG", Name: "Tower Bersama Infrastructure Tbk", SectorID: 11},
	{Code: "LINK", Name: "Link Net Tbk", SectorID: 11},
	{Code: "BIRD", Name: "Blue Bird Tbk", SectorID: 11},
	{Code: "ASSA", Name: "Adi Sarana Armada Tbk", SectorID: 11},
	{Code: "IPCM", Name: "Jasa Armada Indonesia Tbk", SectorID: 11},
	{Code: "CMNP", Name: "Citra Marga Nusaphala Persada Tbk", SectorID: 11},
	{Code: "META", Name: "Nusantara Infrastructure Tbk", SectorID: 11},
	{Code: "BALI", Name: "Bali Towerindo Sentra Tbk", SectorID: 11},
	{Code: "BTEL", Name: "Bakrie Telecom Tbk", SectorID: 11},
	{Code: "CENT", Name: "Centratama Telekomunikasi Indonesia Tbk", SectorID: 11},
	{Code: "FREN", Name: "Smartfren Telecom Tbk", SectorID: 11},
	{Code: "GHON", Name: "Gihon Telekomunikasi Indonesia Tbk", SectorID: 11},
	{Code: "OASA", Name: "Protech Mitra Perkasa Tbk", SectorID: 11},
	{Code: "PTSN", Name: "Sat Nusapersada Tbk", SectorID: 11},
	{Code: "SHID", Name: "Hotel Sahid Jaya International Tbk", SectorID: 11},
	{Code: "SUPR", Name: "Solusi Tunas Pratama Tbk", SectorID: 11},
	{Code: "TGRA", Name: "Terregra Asia Energy Tbk", SectorID: 11},
	{Code: "TRUK", Name: "Guna Timur Raya Tbk", SectorID: 11},
	{Code: "VERN", Name: "Vernando Investama Tbk", SectorID: 11},
	{Code: "KARW", Name: "Karwell Indonesia Tbk", SectorID: 11},
	{Code: "IDPR", Name: "Indonesia Pondasi Raya Tbk", SectorID: 11},
	{Code: "JTPE", Name: "Jasuindo Tiga Perkasa Tbk", SectorID: 11},
	{Code: "TRAM", Name: "Trada Alam Minera Tbk", SectorID: 11},
	{Code: "WEHA", Name: "WEHA Transportasi Indonesia Tbk", SectorID: 11},
	{Code: "HELI", Name: "Helios Logistik Indo Tbk", SectorID: 11},
	{Code: "SAFE", Name: "Steady Safe Tbk", SectorID: 11},
	{Code: "SDMU", Name: "Sidomulyo Selaras Tbk", SectorID: 11},
	{Code: "SMDR", Name: "Samudera Indonesia Tbk", SectorID: 11},
	{Code: "TAXI", Name: "Express Transindo Utama Tbk", SectorID: 11},
	{Code: "TMAS", Name: "Temas Tbk", SectorID: 11},
	{Code: "TRIS", Name: "Trisula International Tbk", SectorID: 11},
	{Code: "WINS", Name: "Wintermar Offshore Marine Tbk", SectorID: 11},
	{Code: "BLTA", Name: "Berlian Laju Tanker Tbk", SectorID: 11},
	{Code: "HITS", Name: "Humpuss Intermoda Transportasi Tbk", SectorID: 11},
	{Code: "MBTO", Name: "Margo City Realty Tbk", SectorID: 11},
	{Code: "NELY", Name: "Pelayaran Nelly Dwi Putri Tbk", SectorID: 11},
	{Code: "PTIS", Name: "Indo Straits Tbk", SectorID: 11},
	{Code: "RIGS", Name: "Rig Tenders Indonesia Tbk", SectorID: 11},
	{Code: "SAPX", Name: "Satria Antaran Prima Tbk", SectorID: 11},
	{Code: "TNCA", Name: "Trimuda Nuansa Citra Tbk", SectorID: 11},
	{Code: "LRNA", Name: "Eka Sari Lorena Transport Tbk", SectorID: 11},
	{Code: "GIAA", Name: "Garuda Indonesia Tbk", SectorID: 11},
	{Code: "CMNT", Name: "Cemindo Gemilang Tbk", SectorID: 11},
	{Code: "CASA", Name: "Capital Financial Indonesia Tbk", SectorID: 11},
	{Code: "PSSI", Name: "Pelita Samudera Shipping Tbk", SectorID: 11},
	{Code: "CANI", Name: "Capitol Nusantara Indonesia Tbk", SectorID: 11},

	{Code: "ARNA", Name: "Arwana Citramulia Tbk", SectorID: 2},
	{Code: "EKAD", Name: "Ekaputra Wijaya Lestari Tbk", SectorID: 2},
	{Code: "GDST", Name: "Gunawan Dianjaya Steel Tbk", SectorID: 2},
	{Code: "SRSN", Name: "Indo Acidatama Tbk", SectorID: 2},
	{Code: "UNIC", Name: "Unggul Indah Cahaya Tbk", SectorID: 2},
	{Code: "FGGA", Name: "Fuji Finance Indonesia Tbk", SectorID: 2},
	{Code: "INCF", Name: "Indo Komoditi Korpora Tbk", SectorID: 2},
	{Code: "FASW", Name: "Fajar Surya Wisesa Tbk", SectorID: 2},
	{Code: "KICI", Name: "Kedaung Indah Can Tbk", SectorID: 2},
	{Code: "CPRO", Name: "Central Proteina Prima Tbk", SectorID: 2},

	{Code: "ERAA", Name: "Erajaya Swasembada Tbk", SectorID: 7},
	{Code: "PANI", Name: "Pratama Abadi Nusa Industri Tbk", SectorID: 7},
	{Code: "BELI", Name: "Global Digital Niaga Tbk", SectorID: 7},
	{Code: "ENVY", Name: "Envy Technologies Indonesia Tbk", SectorID: 7},
	{Code: "NTBR", Name: "Net Visi Media Tbk", SectorID: 7},
	{Code: "NOVA", Name: "Galva Technologies Tbk", SectorID: 7},
	{Code: "UVCR", Name: "Trimegah Karya Pratama Tbk", SectorID: 7},
	{Code: "WIRG", Name: "WIR Asia Tbk", SectorID: 7},
	{Code: "ZYRX", Name: "Zyrexindo Mandiri Buana Tbk", SectorID: 7},
	{Code: "HAIS", Name: "Hasnur Internasional Shipping Tbk", SectorID: 7},
	{Code: "CHIP", Name: "Pelita Teknologi Global Tbk", SectorID: 7},

	{Code: "BAPA", Name: "Bekasi Asri Pemula Tbk", SectorID: 9},
	{Code: "BATA", Name: "Sepatu Bata Tbk", SectorID: 9},
	{Code: "EMDE", Name: "Megapolitan Developments Tbk", SectorID: 9},
	{Code: "GAMA", Name: "Gading Development Tbk", SectorID: 9},
	{Code: "GPRA", Name: "Perdana Gapuraprima Tbk", SectorID: 9},
	{Code: "KOTA", Name: "DMS Propertindo Tbk", SectorID: 9},
	{Code: "PAMG", Name: "Bima Sakti Pertiwi Tbk", SectorID: 9},
	{Code: "REAL", Name: "Repower Asia Indonesia Tbk", SectorID: 9},
	{Code: "RISE", Name: "Jaya Sukses Makmur Sentosa Tbk", SectorID: 9},
	{Code: "ROCK", Name: "Rockfields Properti Indonesia Tbk", SectorID: 9},

	{Code: "AUTO", Name: "Astra Otoparts Tbk", SectorID: 4},
	{Code: "BOLT", Name: "Garuda Metalindo Tbk", SectorID: 4},
	{Code: "GDYR", Name: "Goodyear Indonesia Tbk", SectorID: 4},
	{Code: "INDS", Name: "Indospring Tbk", SectorID: 4},
	{Code: "LPIN", Name: "Multi Prima Sejahtera Tbk", SectorID: 4},
	{Code: "PRAS", Name: "Prima Alloy Steel Universal Tbk", SectorID: 4},
	{Code: "SMSM", Name: "Selamat Sempurna Tbk", SectorID: 4},

	{Code: "BUMI", Name: "Bumi Resources Tbk", SectorID: 3},
	{Code: "PTRO", Name: "Petrosea Tbk", SectorID: 3},
	{Code: "BIPI", Name: "Benakat Integra Tbk", SectorID: 3},
	{Code: "ABMM", Name: "ABM Investama Tbk", SectorID: 3},
	{Code: "APEX", Name: "Apexindo Pratama Duta Tbk", SectorID: 3},
	{Code: "ARTI", Name: "Ratu Prabu Energi Tbk", SectorID: 3},
	{Code: "BORN", Name: "Borneo Lumbung Energi & Metal Tbk", SectorID: 3},
	{Code: "CANI", Name: "Capitol Nusantara Indonesia Tbk", SectorID: 3},
	{Code: "DWGL", Name: "Dwi Guna Laksana Tbk", SectorID: 3},
	{Code: "GTBO", Name: "Garda Tujuh Buana Tbk", SectorID: 3},
	{Code: "JAWA", Name: "Jaya Agra Wattie Tbk", SectorID: 3},
	{Code: "LEAD", Name: "Logindo Samudramakmur Tbk", SectorID: 3},
	{Code: "PKPK", Name: "Perdana Karya Perkasa Tbk", SectorID: 3},
	{Code: "SMRU", Name: "SMR Utama Tbk", SectorID: 3},
	{Code: "TBLA", Name: "Tunas Baru Lampung Tbk", SectorID: 3},

	{Code: "CINT", Name: "Chitose Internasional Tbk", SectorID: 10},
	{Code: "DFAM", Name: "Dafam Property Indonesia Tbk", SectorID: 10},
	{Code: "EPMT", Name: "Enseval Putera Megatrading Tbk", SectorID: 10},
	{Code: "ESSA", Name: "Surya Esa Perkasa Tbk", SectorID: 10},
	{Code: "FORU", Name: "Fortune Indonesia Tbk", SectorID: 10},
	{Code: "HOTL", Name: "Saraswati Griya Lestari Tbk", SectorID: 10},
	{Code: "IKAI", Name: "Intikeramik Alamasri Industri Tbk", SectorID: 10},
	{Code: "INTA", Name: "Intraco Penta Tbk", SectorID: 10},
	{Code: "IPTV", Name: "MNC Digital Entertainment Tbk", SectorID: 10},
	{Code: "JSPT", Name: "Jakarta Setiabudi Internasional Tbk", SectorID: 10},
	{Code: "KPAS", Name: "Cottonindo Ariesta Tbk", SectorID: 10},
	{Code: "LAST", Name: "Laskar Semesta Alam Tbk", SectorID: 10},
	{Code: "MFMI", Name: "Multifiling Mitra Indonesia Tbk", SectorID: 10},
	{Code: "MOLI", Name: "Madusari Murni Indah Tbk", SectorID: 10},
	{Code: "NAYS", Name: "Nayati Indonesia Tbk", SectorID: 10},
	{Code: "PANR", Name: "Panorama Sentrawisata Tbk", SectorID: 10},
	{Code: "POLI", Name: "Pollux Properti Indonesia Tbk", SectorID: 10},
	{Code: "PTSN", Name: "Sat Nusapersada Tbk", SectorID: 10},
	{Code: "RUIS", Name: "Radiant Utama Interinsco Tbk", SectorID: 10},
	{Code: "SKRN", Name: "Superkrane Mitra Utama Tbk", SectorID: 10},
	{Code: "TIRA", Name: "Tira Austenite Tbk", SectorID: 10},
	{Code: "TOYS", Name: "Sunindo Adipersada Tbk", SectorID: 10},
	{Code: "UNSP", Name: "Bakrie Sumatera Plantations Tbk", SectorID: 10},

	{Code: "ASGR", Name: "Astra Graphia Tbk", SectorID: 7},
	{Code: "BSSR", Name: "Baramulti Suksessarana Tbk", SectorID: 7},
	{Code: "KREN", Name: "Kresna Graha Investama Tbk", SectorID: 7},
	{Code: "MARI", Name: "Mahaka Radio Integra Tbk", SectorID: 7},
	{Code: "MBSS", Name: "Mitrabahtera Segara Sejati Tbk", SectorID: 7},
}

var seededSectors = []model.Sector{
	{ID: 1, Name: "Consumer Goods", Slug: "consumer-goods", Description: "Barang konsumsi dan makanan minuman"},
	{ID: 2, Name: "Basic Materials", Slug: "basic-materials", Description: "Bahan dasar, semen, kimia, dan industri"},
	{ID: 3, Name: "Energy & Mining", Slug: "energy-mining", Description: "Energi, minyak, gas, dan pertambangan"},
	{ID: 4, Name: "Automotive", Slug: "automotive", Description: "Otomotif dan komponen"},
	{ID: 5, Name: "Agriculture", Slug: "agriculture", Description: "Agrikultur dan perkebunan"},
	{ID: 6, Name: "Healthcare", Slug: "healthcare", Description: "Kesehatan dan farmasi"},
	{ID: 7, Name: "Technology", Slug: "technology", Description: "Teknologi dan digital"},
	{ID: 8, Name: "Financials", Slug: "financials", Description: "Perbankan, asuransi, dan keuangan"},
	{ID: 9, Name: "Property & Real Estate", Slug: "property-real-estate", Description: "Properti dan real estate"},
	{ID: 10, Name: "Trade & Services", Slug: "trade-services", Description: "Perdagangan, jasa, dan investasi"},
	{ID: 11, Name: "Infrastructure & Utilities", Slug: "infrastructure-utilities", Description: "Infrastruktur, transportasi, dan telekomunikasi"},
}

func (s *IDXScraper) FetchStockList() ([]model.Stock, []model.Sector, error) {
	stocks := make([]model.Stock, len(seededStocks))
	copy(stocks, seededStocks)

	sectors := make([]model.Sector, len(seededSectors))
	copy(sectors, seededSectors)

	return stocks, sectors, nil
}

func (s *IDXScraper) FetchFundamentals(stockCode string) (*model.StockFundamental, error) {
	code := strings.ToUpper(strings.TrimSpace(stockCode))
	if code == "" {
		return nil, fmt.Errorf("idx scraper: empty stock code")
	}

	yahoo := &YahooScraper{}

	fund, err := yahoo.FetchFundamentalsFromYahoo(code + ".JK")
	if err == nil && fund != nil {
		return fund, nil
	}

	// Fallback 2: Financial Modeling Prep (real ratios, if API key configured).
	fmp := NewFMPScraper()
	if fmp.IsConfigured() {
		if fund, err := fmp.FetchFundamentalsFromFMP(code + ".JK"); err == nil && fund != nil {
			return fund, nil
		}
	}

	end := time.Now().In(jakartaLoc)
	start := end.AddDate(-1, 0, 0)

	prices, err := yahoo.FetchHistorical(code+".JK", start, end)
	if err != nil {
		return nil, fmt.Errorf("idx scraper: fetch price for %s: %w", code, err)
	}

	if len(prices) == 0 {
		return nil, fmt.Errorf("idx scraper: no price data for %s", code)
	}

	latest := prices[len(prices)-1]

	var avgClose, avgVolume float64
	var totalVolume float64
	for _, p := range prices {
		avgClose += p.Close
		avgVolume += float64(p.Volume)
	}
	count := float64(len(prices))
	avgClose /= count
	totalVolume = avgVolume
	avgVolume /= count

	estimatedEPS := latest.Close * 0.05
	estimatedBVPS := latest.Close * 0.40
	estimatedRevenue := totalVolume * 1000
	estimatedNetIncome := estimatedRevenue * 0.12
	estimatedAssets := estimatedRevenue * 2.5
	estimatedLiabilities := estimatedAssets * 0.55
	estimatedEquity := estimatedAssets - estimatedLiabilities

	roe := 0.0
	if estimatedEquity > 0 {
		roe = (estimatedNetIncome / estimatedEquity) * 100
	}

	roa := 0.0
	if estimatedAssets > 0 {
		roa = (estimatedNetIncome / estimatedAssets) * 100
	}

	per := 0.0
	if estimatedEPS > 0 {
		per = latest.Close / estimatedEPS
	}

	pbv := 0.0
	if estimatedBVPS > 0 {
		pbv = latest.Close / estimatedBVPS
	}

	der := 0.0
	if estimatedEquity > 0 {
		der = estimatedLiabilities / estimatedEquity
	}

	npm := 0.0
	if estimatedRevenue > 0 {
		npm = (estimatedNetIncome / estimatedRevenue) * 100
	}

	dividendYield := 1.5

	return &model.StockFundamental{
		Period:           "FY2025",
		ReportType:       "annual",
		Source:           "estimated",
		Revenue:          estimatedRevenue,
		NetIncome:        estimatedNetIncome,
		EPS:              estimatedEPS,
		BVPS:             estimatedBVPS,
		TotalAssets:      estimatedAssets,
		TotalLiabilities: estimatedLiabilities,
		Equity:           estimatedEquity,
		ROE:              roe,
		ROA:              roa,
		PER:              per,
		PBV:              pbv,
		DER:              der,
		NetProfitMargin:  npm,
		DividendYield:    dividendYield,
	}, nil
}

func (s *IDXScraper) FetchAllIDXStocks() ([]model.Stock, []model.Sector) {
	stocks := make([]model.Stock, len(seededStocks))
	copy(stocks, seededStocks)

	sectors := make([]model.Sector, len(seededSectors))
	copy(sectors, seededSectors)

	return stocks, sectors
}
