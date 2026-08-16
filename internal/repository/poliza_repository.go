package repository

import (
	"context"

	"github.com/omaradriano/cobranzawebscrapper_server/internal/models"
	"gorm.io/gorm"
)

type PolizaFilters struct {
	AgenteID    int
	Filters     map[string]string
	PageSize    int
	CurrentPage int
}

type PolizaDetails struct {
	Total             int
	Activas           int
	Inactivas         int
	PorVencer         int
	CoberturaActiva   int
	SinPagoRegistrado int
	Recientes         int
}

type BirthdateResult struct {
	NombreCompleto string
	NextBirthday   string
	NumPoliza      string
}

type PolizaRepository interface {
	FindByNumPoliza(ctx context.Context, numPoliza string, agenteID int) (*models.Poliza, error)
	GetNumPolizasByAgenteUUID(ctx context.Context, uuid string) ([]string, error)
	GetDetails(ctx context.Context, agenteUUID string) (*PolizaDetails, error)
	BulkCreate(ctx context.Context, polizas []models.Poliza, asegurados [][]models.Asegurado) ([]int64, []string, error)
	CreateSingle(ctx context.Context, poliza *models.Poliza) error
	FindPolizaIDByNumPoliza(ctx context.Context, numPoliza string) (int, error)
	GetPolizaWithAsegurados(ctx context.Context, numPoliza string, agenteID int) (*models.Poliza, error)
	GetPolizasPaginated(ctx context.Context, filters PolizaFilters) ([]map[string]any, int64, error)
	GetBirthdates(ctx context.Context, agenteID int) ([]BirthdateResult, error)
	UpdateDiaCobro(ctx context.Context, numPoliza string, agenteID int, diaCobro int16) error
}

type polizaRepository struct {
	db *gorm.DB
}

func NewPolizaRepository(db *gorm.DB) PolizaRepository {
	return &polizaRepository{db: db}
}

func (r *polizaRepository) FindByNumPoliza(ctx context.Context, numPoliza string, agenteID int) (*models.Poliza, error) {
	var poliza models.Poliza
	err := r.db.WithContext(ctx).
		Where("numpoliza = ? AND agente_id = ?", numPoliza, agenteID).
		First(&poliza).Error
	if err != nil {
		return nil, err
	}
	return &poliza, nil
}

func (r *polizaRepository) GetNumPolizasByAgenteUUID(ctx context.Context, uuid string) ([]string, error) {
	var results []string
	err := r.db.WithContext(ctx).
		Model(&models.Poliza{}).
		Joins("JOIN agentes a ON polizas.agente_id = a.agente_id").
		Where("a.agente_uuid = ?", uuid).
		Pluck("numpoliza", &results).Error
	return results, err
}

func (r *polizaRepository) GetDetails(ctx context.Context, agenteUUID string) (*PolizaDetails, error) {
	var details PolizaDetails

	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN p.estatus = 'ACTIVO' THEN 1 ELSE 0 END) as activas,
			SUM(CASE WHEN p.estatus = 'INACTIVO' THEN 1 ELSE 0 END) as inactivas,
			SUM(CASE WHEN ppc.next_payment IS NOT NULL AND ppc.next_payment <= NOW() + INTERVAL '30 days' AND p.estatus = 'ACTIVO' THEN 1 ELSE 0 END) as por_vencer,
			SUM(CASE WHEN p.estatus = 'ACTIVO' AND ppc.next_payment IS NOT NULL AND ppc.next_payment > NOW() THEN 1 ELSE 0 END) as cobertura_activa,
			SUM(CASE WHEN ppl.payment_log_id IS NULL THEN 1 ELSE 0 END) as sin_pago_registrado
		FROM polizas p
		JOIN agentes a ON p.agente_id = a.agente_id
		JOIN polizas_payments_conf ppc ON ppc.poliza_id = p.poliza_id
		LEFT JOIN polizas_payments_log ppl ON ppl.poliza_id = p.poliza_id
		WHERE a.agente_uuid = ?`, agenteUUID).
		Scan(&details).Error
	if err != nil {
		return nil, err
	}

	err = r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) as recientes
		FROM polizas p
		JOIN agentes a ON p.agente_id = a.agente_id
		WHERE a.agente_uuid = ? AND p.fecha_emision >= NOW() - INTERVAL '30 days'`, agenteUUID).
		Scan(&details.Recientes).Error
	if err != nil {
		return nil, err
	}

	return &details, nil
}

func (r *polizaRepository) BulkCreate(ctx context.Context, polizas []models.Poliza, asegurados [][]models.Asegurado) ([]int64, []string, error) {
	var ids []int64
	var nums []string

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, poliza := range polizas {
			if err := tx.Create(&poliza).Error; err != nil {
				return err
			}
			ids = append(ids, int64(poliza.PolizaID))
			nums = append(nums, poliza.NumPoliza)

			if i < len(asegurados) {
				for j := range asegurados[i] {
					pid := int64(poliza.PolizaID)
					asegurados[i][j].PolizaID = &pid
				}
				if len(asegurados[i]) > 0 {
					if err := tx.Create(&asegurados[i]).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})

	return ids, nums, err
}

func (r *polizaRepository) CreateSingle(ctx context.Context, poliza *models.Poliza) error {
	return r.db.WithContext(ctx).Create(poliza).Error
}

func (r *polizaRepository) FindPolizaIDByNumPoliza(ctx context.Context, numPoliza string) (int, error) {
	var poliza models.Poliza
	err := r.db.WithContext(ctx).Select("poliza_id").Where("numpoliza = ?", numPoliza).First(&poliza).Error
	if err != nil {
		return 0, err
	}
	return poliza.PolizaID, nil
}

func (r *polizaRepository) GetPolizaWithAsegurados(ctx context.Context, numPoliza string, agenteID int) (*models.Poliza, error) {
	var poliza models.Poliza
	err := r.db.WithContext(ctx).
		Preload("Asegurados").
		Preload("PaymentConf").
		Joins("JOIN agentes a ON polizas.agente_id = a.agente_id").
		Where("polizas.numpoliza = ? AND a.agente_id = ?", numPoliza, agenteID).
		First(&poliza).Error
	if err != nil {
		return nil, err
	}
	return &poliza, nil
}

func (r *polizaRepository) GetPolizasPaginated(ctx context.Context, filters PolizaFilters) ([]map[string]any, int64, error) {
	var totalRecords int64

	baseQuery := r.db.WithContext(ctx).
		Model(&models.Poliza{}).
		Joins("JOIN agentes a ON polizas.agente_id = a.agente_id").
		Joins("JOIN polizas_payments_conf ppc ON ppc.poliza_id = polizas.poliza_id").
		Joins("LEFT JOIN polizas_payments_log ppl ON ppl.poliza_id = polizas.poliza_id").
		Where("a.agente_id = ?", filters.AgenteID)

	for col, val := range filters.Filters {
		switch col {
		case "estatus":
			baseQuery = baseQuery.Where("polizas.estatus = ?", val)
		case "numpoliza":
			baseQuery = baseQuery.Where("polizas.numpoliza ILIKE ?", "%"+val+"%")
		case "next_due":
			baseQuery = baseQuery.Where("ppc.next_payment <= NOW() + INTERVAL '30 days' AND polizas.estatus = 'ACTIVO'")
		case "asegurado":
			baseQuery = baseQuery.
				Joins("JOIN asegurados aseg ON aseg.poliza_id = polizas.poliza_id").
				Where("aseg.nombre_completo ILIKE ?", "%"+val+"%")
		}
	}

	if err := baseQuery.Count(&totalRecords).Error; err != nil {
		return nil, 0, err
	}

	offset := (filters.CurrentPage - 1) * filters.PageSize

	var results []map[string]any
	err := baseQuery.
		Select(`polizas.poliza_id, polizas.dia_cobro, polizas.estatus, polizas.fecha_emision,
			polizas.forma_pago, polizas.medio_cobro, polizas.numpoliza, polizas.plan,
			polizas.tipo_seguro, polizas.addr_calle, polizas.addr_codigopostal,
			polizas.addr_ciudad, polizas.addr_colonia, polizas.addr_estado,
			ppc.next_payment, polizas.moneda, polizas.pais, polizas.telefono,
			polizas.email, polizas.suma_asegurada, polizas.last_modified,
			polizas.poliza_uuid,
			CASE WHEN ppl.payment_log_id IS NOT NULL THEN 'true' ELSE 'false' END as payment_exist`).
		Group("polizas.poliza_id, ppc.payment_conf_id, ppl.payment_log_id").
		Order("polizas.poliza_id DESC").
		Limit(filters.PageSize).
		Offset(offset).
		Find(&results).Error
	if err != nil {
		return nil, 0, err
	}

	return results, totalRecords, err
}

func (r *polizaRepository) UpdateDiaCobro(ctx context.Context, numPoliza string, agenteID int, diaCobro int16) error {
	result := r.db.WithContext(ctx).
		Model(&models.Poliza{}).
		Where("numpoliza = ? AND agente_id = ?", numPoliza, agenteID).
		Update("dia_cobro", diaCobro)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *polizaRepository) GetBirthdates(ctx context.Context, agenteID int) ([]BirthdateResult, error) {
	var results []BirthdateResult
	err := r.db.WithContext(ctx).Raw(`
		WITH birthday_calc AS (
		    SELECT a.nombre_completo, a.birthday, p.numpoliza,
		        MAKE_DATE(EXTRACT(YEAR FROM NOW())::int, EXTRACT(MONTH FROM birthday)::int, EXTRACT(DAY FROM birthday)::int)::timestamp AS has_current_year_birthday
		    FROM asegurados a
		    JOIN polizas p ON p.poliza_id = a.poliza_id
		    JOIN agentes ag ON ag.agente_id = p.agente_id
		    WHERE ag.agente_id = ?
		),
		distinct_birthdays AS (
		    SELECT DISTINCT ON (nombre_completo)
		        nombre_completo,
		        CASE WHEN has_current_year_birthday < NOW() THEN has_current_year_birthday + INTERVAL '1 year' ELSE has_current_year_birthday END AS next_birthday,
		        numpoliza
		    FROM birthday_calc
		    ORDER BY nombre_completo, next_birthday ASC
		)
		SELECT nombre_completo, next_birthday, numpoliza as num_poliza
		FROM distinct_birthdays
		ORDER BY next_birthday ASC`, agenteID).
		Scan(&results).Error
	return results, err
}
