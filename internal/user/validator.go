package user

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

// UserValidator validates user input data
type UserValidator struct{}

// NewUserValidator creates a new validator
func NewUserValidator() *UserValidator {
	return &UserValidator{}
}

// ValidateName validates user name
func (v *UserValidator) ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("نام نمی‌تواند خالی باشد")
	}
	if len(name) < 2 {
		return fmt.Errorf("نام باید حداقل 2 حرف باشد")
	}
	if len(name) > 100 {
		return fmt.Errorf("نام نمی‌تواند بیشتر از 100 حرف باشد")
	}
	// Allow Persian and English characters only
	if !isValidNameFormat(name) {
		return fmt.Errorf("نام شامل کاراکترهای نامعتبر است")
	}
	return nil
}

// ValidateAge validates user age
func (v *UserValidator) ValidateAge(age int) error {
	if age < 13 {
		return fmt.Errorf("سن شما کمتر از حد مجاز است")
	}
	if age > 120 {
		return fmt.Errorf("سن نامعتبر است")
	}
	return nil
}

// ValidateCity validates city name
func (v *UserValidator) ValidateCity(city string) error {
	city = strings.TrimSpace(city)
	if city == "" {
		return fmt.Errorf("نام شهر نمی‌تواند خالی باشد")
	}
	if len(city) > 100 {
		return fmt.Errorf("نام شهر خیلی بلند است")
	}
	return nil
}

// ValidateGender validates gender
func (v *UserValidator) ValidateGender(gender string) error {
	if gender != string(Male) && gender != string(Female) {
		return fmt.Errorf("جنسیت نامعتبر است")
	}
	return nil
}

// ValidateCoordinates validates GPS coordinates
func (v *UserValidator) ValidateCoordinates(latitude, longitude float64) error {
	if latitude < -90 || latitude > 90 {
		return fmt.Errorf("عرض جغرافیایی نامعتبر است")
	}
	if longitude < -180 || longitude > 180 {
		return fmt.Errorf("طول جغرافیایی نامعتبر است")
	}
	return nil
}

// ValidateProvince validates province name
func (v *UserValidator) ValidateProvince(province string) error {
	if !isValidProvince(province) {
		return fmt.Errorf("استان انتخاب شده معتبر نیست")
	}
	return nil
}

// ValidateProfilePhoto validates photo reference
func (v *UserValidator) ValidateProfilePhoto(photoRef string) error {
	photoRef = strings.TrimSpace(photoRef)
	if photoRef == "" {
		return fmt.Errorf("فایل عکس نامعتبر است")
	}
	if len(photoRef) > 255 {
		return fmt.Errorf("مرجع عکس خیلی بلند است")
	}
	return nil
}

// ----------- Helper Functions -----------

// isValidNameFormat checks if name contains only allowed characters
func isValidNameFormat(name string) bool {
	// Allow Persian letters, English letters, spaces, and hyphens
	pattern := `^[\p{L}\s\-']+$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(name)
}

// isValidProvince checks if province is in the valid list
func isValidProvince(province string) bool {
	validProvinces := map[string]bool{
// isValidNameFormat checks if name contains only allowed characters
"آذربایجان شرقی":          true, 
"آذربایجان غربی":          true, 
"اردبیل":          true, 
"اصفهان":          true,
 "البرز":          true,
	"ایلام":          true, 
	"بوشهر":          true, 
	"تهران":          true, 
	"چهارمحال و بختیاری":          true, 
	"خراسان جنوبی":          true,
	"خراسان رضوی":          true, 
	"خراسان شمالی":          true, 
	"خوزستان":          true, 
	"زنجان":          true, 
	"سمنان":          true,
	"سیستان و بلوچستان":          true, 
	"فارس":          true, 
	"قزوین":          true, 
	"قم":          true, 
	"کردستان":          true,
	"کرمان":          true, 
	"کرمانشاه":          true, 
	"کهگیلویه و بویراحمد":          true, 
	"گلستان":          true, 
	"گیلان":          true,
	"لرستان":          true, 
	"مازندران":          true, 
	"مرکزی":          true, 
	"هرمزگان":          true, 
	"همدان":          true, 
	"یزد":          true,


		
	}
	return validProvinces[province]
}

// ValidatePhoneNumber validates phone number format
func (v *UserValidator) ValidatePhoneNumber(phone string) error {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil // optional field
	}
	
	// Basic Persian phone validation
	pattern := `^(\+98|0)9\d{9}$`
	re := regexp.MustCompile(pattern)
	if !re.MatchString(phone) {
		return fmt.Errorf("شماره تلفن نامعتبر است")
	}
	return nil
}

// ValidateBio validates user bio
func (v *UserValidator) ValidateBio(bio string) error {
	if len(bio) > 500 {
		return fmt.Errorf("بیوگرافی نمی‌تواند بیشتر از 500 حرف باشد")
	}
	return nil
}

// ValidateMinAge validates minimum age preference
func (v *UserValidator) ValidateMinAge(age *int) error {
	if age == nil {
		return nil
	}
	if *age < 13 || *age > 120 {
		return fmt.Errorf("حداقل سن نامعتبر است")
	}
	return nil
}

// ValidateMaxAge validates maximum age preference
func (v *UserValidator) ValidateMaxAge(age *int) error {
	if age == nil {
		return nil
	}
	if *age < 13 || *age > 120 {
		return fmt.Errorf("حداکثر سن نامعتبر است")
	}
	return nil
}

// ValidateAgeRange validates age range is logical
func (v *UserValidator) ValidateAgeRange(minAge, maxAge *int) error {
	if minAge == nil || maxAge == nil {
		return nil
	}
	if *minAge > *maxAge {
		return fmt.Errorf("حداقل سن نمی‌تواند بیشتر از حداکثر سن باشد")
	}
	return nil
}
