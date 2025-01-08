import numpy as np
import random
import math
from scipy.optimize import curve_fit, minimize
from functools import partial

data = np.genfromtxt('data.csv', delimiter=',')
concurrency = data[1:,1]
exectime = data[1:,2]

func_mem = 128 #MB 
node_mem = 2048 #MB 
MAX_CONCURRENCY=concurrency.max() #max concurrency level (la definisco a priori nel makefile) 

def rt_model (conc, alpha):
    return np.exp(alpha*conc)

# curve fit
alpha, _ = curve_fit(rt_model, concurrency, exectime) 
rt_model = lambda x: np.exp(alpha*x)  

# Optimal for RT (sappiamo gia' che e' 1 nel data.csv perche' e' un esempio e lo sappiamo a priori ed è il minimo)   
opt_rt = int(minimize(rt_model, x0=MAX_CONCURRENCY, bounds=[(1, MAX_CONCURRENCY)]).x[0])
print(f"Optimal for RT: {opt_rt}")


delta_rt = lambda conc: (rt_model(conc)-rt_model(opt_rt))/rt_model(opt_rt)
opt = int(minimize(delta_rt, x0=MAX_CONCURRENCY, bounds=[(1, MAX_CONCURRENCY)]).x[0])
print(f"Multi-objective sol: {opt}")
