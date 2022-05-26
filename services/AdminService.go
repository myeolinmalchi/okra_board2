package services

import (
	"fmt"
	"okra_board2/models"
	"okra_board2/repositories"
	"okra_board2/utils/encryption"
	"regexp"
)

type AdminService interface {
    
    // db상에 저장된 관리자의 계정 정보와 입력된 정보를 비교하여
    // 일치할경우 true, 그렇지 않을 경우 false를 반환한다.
    Login(*models.Admin)        bool

    // 입력된 신규 관리자 정보의 유효성을 검증하고 db에 저장한다.
    // 유효하지 않은 정보: return (false, 유효성 검사 결과)
    // 유효한 정보이나 db 저장에 실패: return (false, nil)
    // 유효한 정보이며 db 저장에 성공: return (true, nil)
    Register(*models.Admin)     (bool, *models.AdminValidationResult)

    // 입력된 관리자 정보의 유효성을 검증하고 db를 갱신한다.
    // 유효하지 않은 정보: return (false, 유효성 검사 결과)
    // 유효한 정보이나 db 갱신에 실패: return (false, nil)
    // 유효한 정보이며 db 갱신에 성공: return (true, nil)
    Update(*models.Admin)       (bool, *models.AdminValidationResult)
}

type AdminServiceImpl struct {
    adminRepo repositories.AdminRepository
}

func NewAdminServiceImpl(adminRepo repositories.AdminRepository) AdminService {
    return &AdminServiceImpl{ adminRepo: adminRepo }
}

// Validate Admin ID. If valid, it returns nil.
func (s *AdminServiceImpl) checkID(id string) *string {
    var msg string
    if match, _ := regexp.MatchString("^[a-z]+[a-z0-9]{5,19}$", id); !match {
        msg = "유효하지 않은 ID입니다."
    } else if s.adminRepo.CheckAdminExists("id", id) {
        msg = "이미 존재하는 ID입니다."
    } else {
        return nil
    }
    return &msg
}

// Validate Admin PW. If valid, it returns nil.
func (s *AdminServiceImpl) checkPW(pw string) *string {
    var msg string
    if match, _ := regexp.MatchString("^(?=.*\\d)(?=.*[a-zA-Z])[0-9a-zA-Z]{8,16}$", pw); !match {
        msg = "유효하지 않은 PW입니다."
    } else {
        return nil
    }
    return &msg
}

// Validate Admin Name. If valid, it returns nil.
func (s *AdminServiceImpl) checkName(name string) *string {
    var msg string
    if match, _ := regexp.MatchString("^[ㄱ-힣]+$", name); !match {
        msg = "유효하지 않은 이름입니다."
    } else {
        return nil
    }
    return &msg
}

// Validate Admin Email. If valid, it returns nil.
func (s *AdminServiceImpl) checkEmail(email string) *string {
    var msg string
    if match, _ := regexp.
        MatchString("^[0-9a-zA-Z]([-_.]?[0-9a-zA-Z])*@[0-9a-zA-Z]([-_.]?[0-9a-zA-Z])*.[a-zA-Z]{2,3}$", email); !match {
        msg = "유효하지 않은 이메일입니다."
    } else if s.adminRepo.CheckAdminExists("email", email) {
        msg = "이미 사용중인 이메일입니다."
    } else {
        return nil
    }
    return &msg
}

// Validate Admin Phone. If valid, it returns nil.
func (s *AdminServiceImpl) checkPhone(phone string) *string {
    var msg string
    if match, _ := regexp.MatchString("^\\d{3}-\\d{3,4}-\\d{4}$", phone); !match {
        msg = "유효하지 않은 전화번호입니다."
    } else if s.adminRepo.CheckAdminExists("phone", phone) {
        msg = "이미 사용중인 전화번호입니다."
    } else {
        return nil
    }
    return &msg
}

// Validation when regist admin account.
// If valid, it returns nil.
func (s *AdminServiceImpl) adminRegistValidation(admin *models.Admin) *models.AdminValidationResult {
    result := &models.AdminValidationResult {
        ID: s.checkID(admin.ID),
        Password: s.checkPW(admin.Password),
        Email: s.checkEmail(admin.Email),
        Name: s.checkName(admin.Name),
        Phone: s.checkPhone(admin.Phone),
    }
    return result.GetOrNil()
}

// Validation when update admin account
// If valid, it returns nil.
func (s *AdminServiceImpl) adminUpdateValidation(admin *models.Admin) (result *models.AdminValidationResult) {
    result = &models.AdminValidationResult {
        Password: s.checkPW(admin.Password),
        Email: s.checkEmail(admin.Email),
        Name: s.checkName(admin.Name),
        Phone: s.checkPhone(admin.Phone),
    }
    return result.GetOrNil()
}

func (s *AdminServiceImpl) Login(admin *models.Admin) bool {
    insertedPassword := admin.Password
    adminDetail, err := s.adminRepo.GetAdmin(admin.ID)
    if err != nil {
        fmt.Println(err)
        return false
    }
    return encryption.EncryptSHA256(insertedPassword) == adminDetail.Password
}

func (s *AdminServiceImpl) Register(admin *models.Admin) (bool, *models.AdminValidationResult) {
    result := s.adminRegistValidation(admin)
    if result == nil {
        if err := s.adminRepo.InsertAdmin(admin); err != nil {
            return false, nil
        } 
        return true, nil
    }
    return false, result
}

func (s *AdminServiceImpl) Update(admin *models.Admin) (bool, *models.AdminValidationResult) {
    result := s.adminUpdateValidation(admin)
    if result == nil {
        if err := s.adminRepo.UpdateAdmin(admin); err != nil {
            return false, nil
        } 
        return true, nil
    }
    return false, result
}
