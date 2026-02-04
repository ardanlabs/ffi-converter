typedef struct {
    double value;
    int32_t precision;
    uint8_t use_cache;
} CalcConfig;

typedef struct Calc_s* Calc;

CalcConfig calc_default_config(void);
Calc calc_create(CalcConfig config);
void calc_free(Calc calc);
double calc_add(Calc calc, double a, double b);
const char* calc_get_version(void);
int32_t calc_format(Calc calc, char* buf, size_t buf_size);
