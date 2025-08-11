#!/usr/bin/env python3
"""
SUNAT RUC Scraper - Python version
Requires: pip install selenium webdriver-manager
"""

import json
import sys
import time
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.chrome.service import Service
from webdriver_manager.chrome import ChromeDriverManager

class SUNATScraper:
    def __init__(self, headless=False):
        options = webdriver.ChromeOptions()
        if headless:
            options.add_argument('--headless')
        options.add_argument('--no-sandbox')
        options.add_argument('--disable-dev-shm-usage')
        
        self.driver = webdriver.Chrome(
            service=Service(ChromeDriverManager().install()),
            options=options
        )
        self.base_url = "https://e-consultaruc.sunat.gob.pe/cl-ti-itmrconsruc/FrameCriterioBusquedaWeb.jsp"
    
    def close(self):
        self.driver.quit()
    
    def scrape_ruc(self, ruc):
        try:
            # Navigate to the page
            self.driver.get(self.base_url)
            
            # Wait for the page to load
            wait = WebDriverWait(self.driver, 10)
            
            # Enter RUC
            ruc_input = wait.until(EC.presence_of_element_located((By.ID, "txtRuc")))
            ruc_input.clear()
            ruc_input.send_keys(ruc)
            
            # Click search button
            search_button = self.driver.find_element(By.ID, "btnAceptar")
            search_button.click()
            
            # Wait for results
            time.sleep(3)
            
            # Extract information
            info = {"ruc": ruc}
            
            # Find all table cells
            cells = self.driver.find_elements(By.TAG_NAME, "td")
            
            for i in range(0, len(cells) - 1, 2):
                try:
                    label = cells[i].text.strip().lower()
                    value = cells[i + 1].text.strip()
                    
                    if "razón social" in label or "razon social" in label:
                        info["razon_social"] = value
                    elif "tipo contribuyente" in label:
                        info["tipo_contribuyente"] = value
                    elif "nombre comercial" in label:
                        info["nombre_comercial"] = value
                    elif "fecha de inscripción" in label or "fecha de inscripcion" in label:
                        info["fecha_inscripcion"] = value
                    elif "fecha de inicio de actividades" in label:
                        info["fecha_inicio_actividades"] = value
                    elif "estado" in label and "contribuyente" in label:
                        info["estado"] = value
                    elif "condición" in label or "condicion" in label:
                        info["condicion"] = value
                    elif "domicilio fiscal" in label:
                        info["domicilio_fiscal"] = value
                    elif "sistema emisión" in label or "sistema emision" in label:
                        info["sistema_emision"] = value
                    elif "actividad comercio exterior" in label:
                        info["actividad_comercio_exterior"] = value
                    elif "sistema contabilidad" in label:
                        info["sistema_contabilidad"] = value
                except:
                    continue
            
            return info
            
        except Exception as e:
            print(f"Error scraping RUC {ruc}: {str(e)}")
            return None

def main():
    rucs = ["20606316977"]
    
    if len(sys.argv) > 1:
        rucs = sys.argv[1:]
    
    scraper = SUNATScraper(headless=False)
    
    try:
        for ruc in rucs:
            print(f"Scraping RUC: {ruc}")
            
            info = scraper.scrape_ruc(ruc)
            
            if info:
                # Print JSON
                print(json.dumps(info, indent=2, ensure_ascii=False))
                
                # Save to file
                filename = f"ruc_{ruc}.json"
                with open(filename, 'w', encoding='utf-8') as f:
                    json.dump(info, f, indent=2, ensure_ascii=False)
                
                print(f"Data saved to {filename}\n")
    
    finally:
        scraper.close()

if __name__ == "__main__":
    main()